package dsutil

func LimitSlice[T any](slice []T, limit int) ([]T, int) {
	l := len(slice)

	if l <= limit {
		return slice, 0
	}

	return slice[0:limit], l - limit
}

func FilterSlice[T any](in []T, fn func(T) bool) []T {
	result := []T{}

	for i := range in {
		if fn(in[i]) {
			result = append(result, in[i])
		}
	}

	return result
}
