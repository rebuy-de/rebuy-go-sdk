package dsutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	toJSON := func(t *testing.T, set Set[string]) string {
		data, err := json.Marshal(set)
		assert.NoError(t, err)
		return string(data)
	}

	fromJSON := func(t *testing.T, data string) Set[string] {
		var set Set[string]
		err := json.Unmarshal([]byte(data), &set)
		assert.NoError(t, err)
		return set
	}

	t.Run("Simple", func(t *testing.T) {
		var set Set[string]
		set.Add("foo")
		set.Add("bar")
		set.Add("bar")
		assert.Equal(t, `["bar","foo"]`, toJSON(t, set))
	})

	t.Run("JSON", func(t *testing.T) {
		in := `["bar","foo"]`
		set := fromJSON(t, `["bar","foo"]`)
		out := toJSON(t, set)

		assert.Equal(t, in, out)
	})
}

func TestSetUnion(t *testing.T) {
	cases := []struct {
		Name string
		A, B *Set[string]
		Want *Set[string]
	}{
		{
			Name: "Simple",
			A:    NewSet("a", "b", "c"),
			B:    NewSet("c", "d", "e"),
			Want: NewSet("a", "b", "c", "d", "e"),
		},
		{
			Name: "NilB",
			A:    NewSet("a", "b", "c"),
			B:    nil,
			Want: NewSet("a", "b", "c"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			have := SetUnion(tc.A, tc.B)
			require.Equal(t, tc.Want.ToList(), have.ToList())
		})
	}
}

func TestSetIntersection(t *testing.T) {
	cases := []struct {
		Name string
		A, B *Set[string]
		Want *Set[string]
	}{
		{
			Name: "Simple",
			A:    NewSet("a", "b", "c", "d"),
			B:    NewSet("c", "d", "e"),
			Want: NewSet("c", "d"),
		},
		{
			Name: "NilB",
			A:    NewSet("a", "b", "c"),
			B:    nil,
			Want: NewSet[string](),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			have := SetIntersect(tc.A, tc.B)
			require.Equal(t, tc.Want.ToList(), have.ToList())
		})
	}
}
