package chutil

import (
	"time"

	chgo "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pkg/errors"
)

// Default batcher tuning, applied unless overridden via options.
const (
	defaultMaxSize = 2000
	defaultMaxWait = 2 * time.Second
)

// Addr is the ClickHouse server address, for dependency injection.
type Addr string

// Auth holds ClickHouse credentials. The tags let it be loaded straight from
// Vault via vaultutil.DecodeSecret.
type Auth struct {
	Username string `vault:"clickhouse_username"`
	Password string `vault:"clickhouse_password"`
	Database string `vault:"clickhouse_database"`
}

// config carries the connection options and batcher tuning that the functional
// options mutate before New builds the connection and Batcher.
type config struct {
	options     chgo.Options
	maxSize     int
	maxWait     time.Duration
	sendTimeout time.Duration
}

// Option overrides a connection or batcher default passed to New.
type Option func(*config)

// WithCompression sets the wire compression method (default LZ4).
func WithCompression(method chgo.CompressionMethod) Option {
	return func(c *config) {
		c.options.Compression = &chgo.Compression{Method: method}
	}
}

// WithDialTimeout sets the connection dial timeout (default 5s).
func WithDialTimeout(d time.Duration) Option {
	return func(c *config) {
		c.options.DialTimeout = d
	}
}

// WithMaxOpenConns sets the maximum number of open connections (default 4).
func WithMaxOpenConns(n int) Option {
	return func(c *config) {
		c.options.MaxOpenConns = n
	}
}

// WithMaxIdleConns sets the maximum number of idle connections (default 2).
func WithMaxIdleConns(n int) Option {
	return func(c *config) {
		c.options.MaxIdleConns = n
	}
}

// WithConnMaxLifetime sets how long a connection may be reused (default 1h).
func WithConnMaxLifetime(d time.Duration) Option {
	return func(c *config) {
		c.options.ConnMaxLifetime = d
	}
}

// WithMaxSize sets the row count that triggers a flush (default 2000).
func WithMaxSize(n int) Option {
	return func(c *config) {
		c.maxSize = n
	}
}

// WithMaxWait sets the interval that triggers a flush even below maxSize
// (default 2s).
func WithMaxWait(d time.Duration) Option {
	return func(c *config) {
		c.maxWait = d
	}
}

// WithSendTimeout bounds a single bulk insert (default 30s).
func WithSendTimeout(d time.Duration) Option {
	return func(c *config) {
		c.sendTimeout = d
	}
}

// WithRawOptions is an escape hatch to tweak any chgo.Options field not exposed
// by a dedicated option. It runs after all other options.
func WithRawOptions(fn func(*chgo.Options)) Option {
	return func(c *config) {
		fn(&c.options)
	}
}

// New opens a ClickHouse connection using the native protocol and wraps it in a
// Batcher. It does not block on a healthy server: if the cluster is unreachable,
// Add still buffers and Run retries the next batch send. The insertSQL must be a
// column-qualified INSERT statement, e.g. `INSERT INTO db.table (col_a, col_b)`.
func New[T any](addr Addr, auth Auth, insertSQL string, opts ...Option) (*Batcher[T], error) {
	cfg := config{
		options: chgo.Options{
			Addr: []string{string(addr)},
			Auth: chgo.Auth{
				Database: auth.Database,
				Username: auth.Username,
				Password: auth.Password,
			},
			Compression:     &chgo.Compression{Method: chgo.CompressionLZ4},
			DialTimeout:     5 * time.Second,
			MaxOpenConns:    4,
			MaxIdleConns:    2,
			ConnMaxLifetime: time.Hour,
		},
		maxSize:     defaultMaxSize,
		maxWait:     defaultMaxWait,
		sendTimeout: defaultSendTimeout,
	}
	for _, o := range opts {
		o(&cfg)
	}

	conn, err := chgo.Open(&cfg.options)
	if err != nil {
		return nil, errors.Wrap(err, "open clickhouse connection")
	}

	return newBatcher[T](conn, insertSQL, cfg.maxSize, cfg.maxWait, cfg.sendTimeout), nil
}
