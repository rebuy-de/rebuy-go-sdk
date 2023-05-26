package typeutil

import (
	"context"
	"fmt"
)

func FromContext[T any](ctx context.Context, key string) *T {
	raw := ctx.Value(key)
	if raw == nil {
		return nil
	}

	typed, ok := raw.(*T)
	if !ok {
		return nil
	}

	return typed
}

func FromContextSingleton[T any](ctx context.Context) *T {
	var name *T
	return FromContext[T](ctx, fmt.Sprintf("singleton::%T", name))
}

func ContextWithValue[T any](ctx context.Context, key string, value *T) context.Context {
	return context.WithValue(ctx, key, value)
}

func ContextWithValueSingleton[T any](ctx context.Context, value *T) context.Context {
	return ContextWithValue(ctx, fmt.Sprintf("singleton::%T", value), value)
}
