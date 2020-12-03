package redisutil

import "path"

type Prefix string

func (p Prefix) Key(elem ...string) string {
	elem = append([]string{string(p)}, elem...)
	return path.Join(elem...)
}

func (p Prefix) Add(elem ...string) Prefix {
	return Prefix(p.Key(elem...))
}

func (p Prefix) Keys(list []string) []string {
	result := make([]string, len(list))

	for i := 0; i < len(list); i++ {
		result[i] = p.Key(list[i])
	}

	return result
}
