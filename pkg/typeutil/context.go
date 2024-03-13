package typeutil

import (
	"context"
	"fmt"
)

func FromContext[T any](ctx context.Context, key any) *T {
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

type singletonKey string

func getSingletonKey[T any]() singletonKey {
	var dummy *T
	var name = fmt.Sprintf("%T", dummy)
	return singletonKey(name)
}

func FromContextSingleton[T any](ctx context.Context) *T {
	return FromContext[T](ctx, getSingletonKey[T]())
}

func ContextWithValueSingleton[T any](ctx context.Context, value *T) context.Context {
	return context.WithValue(ctx, getSingletonKey[T](), value)
}
