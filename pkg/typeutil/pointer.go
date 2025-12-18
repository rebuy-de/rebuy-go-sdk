package typeutil

// Pointer creates a pointer from the given value. This is useful, when
// creating a pointer from a loop variable [1] or when creating a pointer from
// a return value of another function, since both would require assigning the
// value to an intermediate variable first.
//
// [1]: https://github.com/golang/go/discussions/56010
func Pointer[T any](v T) *T {
	return &v
}

// Value returns the value from a pointer. If the pointer is nil, it will
// return the zero value instead.
func Value[T any](p *T) T {
	return Coalesce(Zero[T](), p)
}

// Coalesce returns the value of the first non-nil pointer from the given
// list of pointers. If all pointers are nil, it returns the fallback value.
// This is useful for providing default values when working with optional
// pointer fields, allowing you to chain multiple fallback options.
func Coalesce[T any](fallback T, pointer ...*T) T {
	for _, p := range pointer {
		if p != nil {
			return *p
		}
	}

	return fallback
}

// CoalesceZero returns the value of the first non-nil pointer from the given
// list of pointers. If all pointers are nil, it returns the zero value for
// type T. This is a convenience wrapper around Coalesce that uses the zero
// value as the fallback.
func CoalesceZero[T any](pointers ...*T) T {
	return Coalesce(Zero[T](), pointers...)
}

// Zero returns the zero value for type T. This is useful when you need to
// explicitly reference a zero value in generic code.
func Zero[T any]() T {
	var zero T
	return zero
}
