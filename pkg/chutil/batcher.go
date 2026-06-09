package chutil

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/runutil"
)

// defaultSendTimeout bounds a single bulk insert so a stuck server cannot block
// the worker indefinitely.
const defaultSendTimeout = 30 * time.Second

// Conn is the subset of driver.Conn the Batcher needs. A real clickhouse-go
// connection satisfies it, and tests can provide a fake.
type Conn interface {
	PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error)
}

// Batcher buffers rows of type T and flushes them to ClickHouse in bulk,
// triggered either by reaching maxSize or by the maxWait interval elapsing. It
// satisfies runutil.WorkerConfiger via Workers, so it can be registered directly
// with runutil.ProvideWorker:
//
//	b, err := chutil.New[Row](addr, auth, insertSQL)
//	runutil.ProvideWorker(c, func() *chutil.Batcher[Row] { return b })
//
// Run is a long-lived loop, so it must not be wrapped in runutil.Repeat.
//
// Any struct with `ch:"..."` tags works as T, because the rows are sent via
// driver.Batch.AppendStruct.
type Batcher[T any] struct {
	conn        Conn
	insertSQL   string
	maxSize     int
	maxWait     time.Duration
	sendTimeout time.Duration

	rows    chan T
	dropped atomic.Uint64
}

// NewWithConn creates a Batcher from an already-opened connection. Prefer New,
// which opens and configures the connection for you; use NewWithConn when you
// need to inject a pre-built or fake Conn (e.g. in tests). The insertSQL must be
// a column-qualified INSERT statement, e.g. `INSERT INTO db.table (col_a, col_b)`.
func NewWithConn[T any](conn Conn, insertSQL string, maxSize int, maxWait time.Duration) *Batcher[T] {
	return newBatcher[T](conn, insertSQL, maxSize, maxWait, defaultSendTimeout)
}

// newBatcher is the shared constructor behind New and NewWithConn.
func newBatcher[T any](conn Conn, insertSQL string, maxSize int, maxWait, sendTimeout time.Duration) *Batcher[T] {
	return &Batcher[T]{
		conn:        conn,
		insertSQL:   insertSQL,
		maxSize:     maxSize,
		maxWait:     maxWait,
		sendTimeout: sendTimeout,
		// The buffer holds two batches so producers can keep enqueueing while a
		// flush is in flight.
		rows: make(chan T, maxSize*2),
	}
}

// Workers satisfies runutil.WorkerConfiger so the Batcher can be registered
// directly with runutil.ProvideWorker. It returns itself as a single long-lived
// worker; the subsystem name is derived from the Batcher type. Run must not be
// wrapped in runutil.Repeat.
func (b *Batcher[T]) Workers() []runutil.Worker {
	return []runutil.Worker{b}
}

// Add enqueues a row. It never blocks: when the buffer is full the row is
// dropped and counted, so producers (e.g. request handlers) are never slowed
// down by a slow ClickHouse.
func (b *Batcher[T]) Add(row T) {
	select {
	case b.rows <- row:
	default:
		b.dropped.Add(1)
	}
}

// Dropped returns the number of rows dropped so far because the buffer was
// full.
func (b *Batcher[T]) Dropped() uint64 {
	return b.dropped.Load()
}

// Run consumes the queue until the context is cancelled, batching rows to
// ClickHouse. On cancellation it drains the already-queued rows and flushes a
// final partial batch before returning. Transient send errors are logged at
// WARN rather than returned, so a single failed batch does not tear down
// sibling workers or page on a self-recovering sink.
func (b *Batcher[T]) Run(ctx context.Context) error {
	ticker := time.NewTicker(b.maxWait)
	defer ticker.Stop()

	var buffer []T
	flush := func() {
		if len(buffer) == 0 {
			return
		}

		err := b.sendBatch(ctx, buffer)
		if err != nil {
			logutil.Get(ctx).Warn("failed to send clickhouse batch", "error", err)
		}

		buffer = buffer[:0]
	}

	for {
		select {
		case <-ctx.Done():
			// Drain whatever is already queued so the final partial batch is
			// not lost. The channel is intentionally not closed: Add may still
			// be called and must not panic.
			for {
				select {
				case row := <-b.rows:
					buffer = append(buffer, row)
					if len(buffer) >= b.maxSize {
						flush()
					}
				default:
					flush()
					if dropped := b.dropped.Load(); dropped > 0 {
						logutil.Get(ctx).Warn("dropped clickhouse rows due to full buffer", "count", dropped)
					}
					return nil
				}
			}
		case row := <-b.rows:
			buffer = append(buffer, row)
			if len(buffer) >= b.maxSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (b *Batcher[T]) sendBatch(ctx context.Context, rows []T) error {
	// Detach from ctx cancellation so a flush triggered during shutdown still
	// completes, while keeping a timeout as a backstop.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), b.sendTimeout)
	defer cancel()

	batch, err := b.conn.PrepareBatch(ctx, b.insertSQL)
	if err != nil {
		return errors.Wrap(err, "prepare batch")
	}

	for i := range rows {
		err = batch.AppendStruct(&rows[i])
		if err != nil {
			// Abort the partially-built batch to free server-side resources,
			// then propagate so the whole batch is dropped instead of silently
			// inserting only the rows up to this point.
			_ = batch.Abort()
			return errors.Wrapf(err, "append row %d", i)
		}
	}

	err = batch.Send()
	if err != nil {
		return errors.Wrap(err, "send batch")
	}

	return nil
}
