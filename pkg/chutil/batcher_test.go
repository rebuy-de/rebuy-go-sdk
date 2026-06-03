package chutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRow struct {
	Value int `ch:"value"`
}

// fakeBatch records appended rows and whether it was sent or aborted. It
// optionally fails on AppendStruct to exercise the abort path.
type fakeBatch struct {
	mu        sync.Mutex
	rows      int
	sent      bool
	aborted   bool
	appendErr error
}

func (b *fakeBatch) Abort() error          { b.mu.Lock(); defer b.mu.Unlock(); b.aborted = true; return nil }
func (b *fakeBatch) Append(v ...any) error { return nil }

func (b *fakeBatch) AppendStruct(v any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appendErr != nil {
		return b.appendErr
	}
	b.rows++
	return nil
}

func (b *fakeBatch) Send() error { b.mu.Lock(); defer b.mu.Unlock(); b.sent = true; return nil }

func (b *fakeBatch) Column(int) driver.BatchColumn { return nil }
func (b *fakeBatch) Flush() error                  { return nil }
func (b *fakeBatch) IsSent() bool                  { return false }
func (b *fakeBatch) Rows() int                     { return b.rows }
func (b *fakeBatch) Columns() []column.Interface   { return nil }
func (b *fakeBatch) Close() error                  { return nil }

// fakeConn hands out a fresh fakeBatch per PrepareBatch call and keeps them all
// for inspection.
type fakeConn struct {
	mu        sync.Mutex
	batches   []*fakeBatch
	appendErr error
}

func (c *fakeConn) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b := &fakeBatch{appendErr: c.appendErr}
	c.batches = append(c.batches, b)
	return b, nil
}

func (c *fakeConn) sentBatches() []*fakeBatch {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []*fakeBatch
	for _, b := range c.batches {
		b.mu.Lock()
		if b.sent {
			out = append(out, b)
		}
		b.mu.Unlock()
	}
	return out
}

const testSQL = "INSERT INTO test.rows (value)"

func TestBatcherCountFlush(t *testing.T) {
	conn := &fakeConn{}
	b := New[testRow](conn, testSQL, 3, time.Hour) // long maxWait so only count triggers

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- b.Run(ctx) }()

	for i := 0; i < 3; i++ {
		b.Add(testRow{Value: i})
	}

	require.Eventually(t, func() bool {
		return len(conn.sentBatches()) == 1
	}, time.Second, 5*time.Millisecond)

	assert.Equal(t, 3, conn.sentBatches()[0].Rows())

	cancel()
	require.NoError(t, <-done)
}

func TestBatcherTimeFlush(t *testing.T) {
	conn := &fakeConn{}
	b := New[testRow](conn, testSQL, 1000, 20*time.Millisecond) // big maxSize so only time triggers

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- b.Run(ctx) }()

	b.Add(testRow{Value: 1})
	b.Add(testRow{Value: 2})

	require.Eventually(t, func() bool {
		sent := conn.sentBatches()
		return len(sent) >= 1 && sent[0].Rows() == 2
	}, time.Second, 5*time.Millisecond)

	cancel()
	require.NoError(t, <-done)
}

func TestBatcherDrainOnCancel(t *testing.T) {
	conn := &fakeConn{}
	b := New[testRow](conn, testSQL, 1000, time.Hour) // neither count nor time triggers before cancel

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- b.Run(ctx) }()

	b.Add(testRow{Value: 1})
	b.Add(testRow{Value: 2})

	// Give Run a moment to consume the queued rows into its buffer, then cancel
	// to trigger the drain-and-flush path.
	require.Eventually(t, func() bool {
		return len(b.rows) == 0
	}, time.Second, 5*time.Millisecond)
	cancel()

	require.NoError(t, <-done)

	sent := conn.sentBatches()
	require.Len(t, sent, 1)
	assert.Equal(t, 2, sent[0].Rows())
}

func TestBatcherDropOnFull(t *testing.T) {
	conn := &fakeConn{}
	// maxSize 2 => buffer capacity 4. Do not Run, so nothing is consumed and the
	// buffer fills up.
	b := New[testRow](conn, testSQL, 2, time.Hour)

	for i := 0; i < 10; i++ {
		b.Add(testRow{Value: i}) // must never block
	}

	assert.Equal(t, uint64(10-4), b.Dropped())
}

func TestBatcherAppendErrorAborts(t *testing.T) {
	conn := &fakeConn{appendErr: errors.New("boom")}
	b := New[testRow](conn, testSQL, 2, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- b.Run(ctx) }()

	b.Add(testRow{Value: 1})
	b.Add(testRow{Value: 2})

	require.Eventually(t, func() bool {
		conn.mu.Lock()
		defer conn.mu.Unlock()
		if len(conn.batches) == 0 {
			return false
		}
		batch := conn.batches[0]
		batch.mu.Lock()
		defer batch.mu.Unlock()
		return batch.aborted
	}, time.Second, 5*time.Millisecond)

	conn.mu.Lock()
	batch := conn.batches[0]
	conn.mu.Unlock()
	batch.mu.Lock()
	assert.True(t, batch.aborted)
	assert.False(t, batch.sent)
	batch.mu.Unlock()

	cancel()
	require.NoError(t, <-done)
}
