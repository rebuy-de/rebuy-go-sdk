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
	if p != nil {
		return *p
	}

	var zero T
	return zero
}
