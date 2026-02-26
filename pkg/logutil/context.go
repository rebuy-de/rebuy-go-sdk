package logutil

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/gosimple/slug"
	"github.com/mitchellh/mapstructure"
)

type contextKey string

const (
	contextKeyMeta contextKey = "meta"
)

// meta is a struct that is stored in the context. It stores the actual logger
// and the trace. The trace is stored separately to be able to recreate the
// logger with a full tracing path.
type meta struct {
	path []trace
	log  *slog.Logger
}

func (m meta) subsystem() string {
	subsystems := []string{"/"}

	for _, t := range m.path {
		subsystems = append(subsystems, t.subsystem)
	}

	return path.Join(subsystems...)
}

type trace struct {
	id        string
	subsystem string
}

// Get extracts the current logger from the given context. It returns the
// default logger, if there is no logger in the context.
func Get(ctx context.Context) *slog.Logger {
	m, ok := ctx.Value(contextKeyMeta).(meta)
	if !ok {
		return slog.Default()
	}
	return m.log
}

// GetSubsystem extracts the name of the subsystem from the given context.
func GetSubsystem(ctx context.Context) string {
	m, ok := ctx.Value(contextKeyMeta).(meta)
	if !ok {
		return ""
	}
	return m.subsystem()
}

// Start creates a new logger and stores it in the returned context.
// Additionally it creates a new trace ID and injects them into the new logger
// together with previous trace IDs from the given context.
func Start(ctx context.Context, subsystem string, opts ...ContextOption) context.Context {
	m, ok := ctx.Value(contextKeyMeta).(meta)
	if !ok {
		m = meta{}
	}

	m.log = slog.Default()
	m.path = append(m.path, trace{
		id:        randomString(12),
		subsystem: subsystem,
	})

	ids := []string{}

	for _, t := range m.path {
		name := fmt.Sprintf("trace-id-%s", slug.Make(t.subsystem))
		m.log = m.log.With(name, t.id)
		ids = append(ids, t.id)
	}

	m.log = m.log.With("subsystem", m.subsystem())
	m.log = m.log.With("trace-id", strings.Join(ids, "-"))

	for _, opt := range opts {
		m = opt(m)
	}

	return context.WithValue(ctx, contextKeyMeta, m)
}

// Update creates a new context with an updated logger.
func Update(ctx context.Context, opts ...ContextOption) context.Context {
	m, ok := ctx.Value(contextKeyMeta).(meta)
	if !ok {
		// This is a wrong usage, but not important enough to add error handling
		// or crash the application. Therefore silently return unaltered
		// context.
		return ctx
	}

	for _, opt := range opts {
		m = opt(m)
	}

	return context.WithValue(ctx, contextKeyMeta, m)
}

// ContextOption is used for modifying a logger.
type ContextOption func(meta) meta

// Field is a ContextOption that sets a single field to the logger.
func Field(key string, value any) ContextOption {
	return func(m meta) meta {
		m.log = m.log.With(key, value)
		return m
	}
}

// WithField is a shortcut for using the Update function with a single Field
// option.
func WithField(ctx context.Context, key string, value any) context.Context {
	return Update(ctx, Field(key, value))
}

// Fields is a ContextOption that sets the given fields to the logger.
func Fields(fields map[string]any) ContextOption {
	return func(m meta) meta {
		attrs := make([]any, 0, len(fields)*2)
		for k, v := range fields {
			attrs = append(attrs, k, v)
		}
		m.log = m.log.With(attrs...)
		return m
	}
}

// WithFields is a shortcut for using the Update function with a single Fields
// option.
func WithFields(ctx context.Context, fields map[string]any) context.Context {
	return Update(ctx, Fields(fields))
}

// FromStruct converts any struct into a valid map[string]any for use as log fields.
// It can be customized with the logfield annotation:
//
//	type Instance struct {
//	    InstanceID   string `logfield:"instance-id"`
//	    InstanceName string `logfield:"instance-name"`
//	}
//
// See mapstructure docs for more information:
// https://pkg.go.dev/github.com/mitchellh/mapstructure?tab=doc
func FromStruct(s any) map[string]any {
	fields := map[string]any{}
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "logfield",
		Result:  &fields,
	})
	if err != nil {
		return map[string]any{"logfield-error": err}
	}

	err = dec.Decode(s)
	if err != nil {
		return map[string]any{"logfield-error": err}
	}

	return fields
}

// PrettyPrint prints the given struct in a readable form. It tries JSON first,
// and if it fails it falls back to fmt.Sprintf.
func PrettyPrint(v any) string {
	raw, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return fmt.Sprintf("%#v", v)
	}

	return string(raw)
}
