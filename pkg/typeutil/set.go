package typeutil

import (
	"encoding/json"
	"fmt"
	"sort"

	"golang.org/x/exp/constraints"
)

// Set implements a set data structure based on built-in maps. This is not
// optimized for large data, but to be convient.
type Set[T constraints.Ordered] struct {
	data map[T]struct{}
}

// NewSet initializes a new Set with the given values. If there are no values
// provides this is equivalent to new(Set[T]).
func NewSet[T constraints.Ordered](values ...T) *Set[T] {
	s := new(Set[T])
	for i := range values {
		s.Add(values[i])
	}
	return s
}

// Add puts a single value to the set. The set will be the same, if it already
// contains the value.
func (s *Set[T]) Add(value T) {
	if s.data == nil {
		s.data = map[T]struct{}{}
	}

	s.data[value] = struct{}{}
}

// Contains returns true, if the given value is part of the set.
func (s *Set[T]) Contains(value T) bool {
	if s == nil || s.data == nil {
		return false
	}

	_, found := s.data[value]
	return found
}

// Remove removes the given value from the set. The set will be the same, if
// the value is not part of it.
func (s *Set[T]) Remove(value T) {
	if s == nil || s.data == nil {
		return
	}

	delete(s.data, value)
}

// Substract removes every element from the given set from the set.
func (s *Set[T]) Subtract(other *Set[T]) {
	if s == nil || s.data == nil {
		return
	}

	for value := range other.data {
		delete(s.data, value)
	}
}

// Len returns the number of all values in the set.
func (s *Set[T]) Len() int {
	if s == nil || s.data == nil {
		return 0
	}
	return len(s.data)
}

// ToList converts the set into a slice. Since the set uses a map as an
// underlying data structure, this will copy each value. So it might be memory
// intensive. Also it sorts the slice to ensure a cosistent result.
func (s *Set[T]) ToList() []T {
	if s == nil || s.data == nil {
		return nil
	}

	list := make([]T, 0, len(s.data))

	if len(s.data) > 0 {
		for v := range s.data {
			list = append(list, v)
		}
	}

	sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })

	return list
}

// AddSet adds each value from the given set to the set.
func (s *Set[T]) AddSet(other *Set[T]) {
	if other == nil {
		return
	}

	for o := range other.data {
		s.Add(o)
	}
}

// MarshalJSON adds support for mashaling the set into a JSON list.
func (s Set[T]) MarshalJSON() ([]byte, error) {
	list := s.ToList()
	return json.Marshal(list)
}

// MarshalJSON adds support for unmashaling the set from a JSON list.
func (s *Set[T]) UnmarshalJSON(data []byte) error {
	list := []T{}
	err := json.Unmarshal(data, &list)
	if err != nil {
		return fmt.Errorf("unmarshal set: %w", err)
	}

	for _, v := range list {
		s.Add(v)
	}

	return nil
}

func SetUnion[T constraints.Ordered](sets ...*Set[T]) *Set[T] {
	result := new(Set[T])

	for s := range sets {
		result.AddSet(sets[s])
	}

	return result
}

// SetIntersect returns a set that only contains elements which exist in all
// sets.
func SetIntersect[T constraints.Ordered](sets ...*Set[T]) *Set[T] {
	result := new(Set[T])

	for _, s := range sets {
		if s == nil || s.data == nil {
			return result
		}
	}

	if len(sets) == 0 {
		return result
	}

	result.AddSet(sets[0])

	for _, s := range sets[1:] {
		for e := range result.data {
			if !s.Contains(e) {
				delete(result.data, e)
			}
		}
	}

	return result
}
