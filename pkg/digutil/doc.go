// Package digutil provides helpers for working with Uber's dig dependency injection library.
//
// # Dependency Injection with digutil
//
// The SDK uses Uber's dig library for dependency injection and provides helpers in this package.
//
// ## Using Parameter Objects for Optional Dependencies
//
// The digutil package provides helpers for optional dependencies:
//
//	// Define options for a service
//	type ServiceOptions struct {
//	    // Required options
//	    Database *sql.DB
//
//	    // Optional options with defaults
//	    CacheTTL time.Duration `optional:"true"`
//	    MaxConns int           `optional:"true"`
//	    Logger   *log.Logger   `optional:"true"`
//	}
//
//	// Service constructor using options
//	func NewService(options ServiceOptions) *Service {
//	    // Apply defaults for optional parameters
//	    if options.CacheTTL == 0 {
//	        options.CacheTTL = 5 * time.Minute
//	    }
//
//	    if options.MaxConns == 0 {
//	        options.MaxConns = 10
//	    }
//
//	    if options.Logger == nil {
//	        options.Logger = log.Default()
//	    }
//
//	    return &Service{
//	        db:       options.Database,
//	        cacheTTL: options.CacheTTL,
//	        maxConns: options.MaxConns,
//	        logger:   options.Logger,
//	    }
//	}
package digutil