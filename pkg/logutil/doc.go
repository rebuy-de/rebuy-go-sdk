// Package logutil provides utilities for structured logging and context-aware logging.
//
// This package enables tracing and correlation of log entries across multiple
// subsystems by maintaining and propagating trace IDs through context.
//
// Main features:
//   - Context-aware logging with automatic trace ID generation
//   - Structured logging with logrus integration
//   - Subsystem path tracking for hierarchical services
//   - Helper methods for adding fields to loggers
//   - Converting structs to log fields with custom field name mapping
//
// Usage:
//
//	ctx = logutil.Start(ctx, "my-subsystem")
//	log := logutil.Get(ctx)
//	log.Info("service started")
//
//	// Add fields to context and logger
//	ctx = logutil.WithField(ctx, "user-id", "12345")
//
//	// Extract subsystem path
//	subsystem := logutil.GetSubsystem(ctx)
//
// The package automatically generates and tracks trace IDs, making it easier to
// follow request flows across multiple components.
//
// Note: Functions invoked from webutil or runutil already have a subsystem and do not need to be started again.
package logutil
