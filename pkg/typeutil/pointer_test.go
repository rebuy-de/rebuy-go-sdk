package typeutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPointer(t *testing.T) {
	cases := []struct {
		Name     string
		TestFunc func(t *testing.T)
	}{
		{
			Name: "Int",
			TestFunc: func(t *testing.T) {
				value := 42
				ptr := Pointer(value)
				require.NotNil(t, ptr)
				require.Equal(t, value, *ptr)
			},
		},
		{
			Name: "String",
			TestFunc: func(t *testing.T) {
				value := "hello"
				ptr := Pointer(value)
				require.NotNil(t, ptr)
				require.Equal(t, value, *ptr)
			},
		},
		{
			Name: "Struct",
			TestFunc: func(t *testing.T) {
				type TestStruct struct {
					X int
					Y string
				}
				value := TestStruct{X: 10, Y: "test"}
				ptr := Pointer(value)
				require.NotNil(t, ptr)
				require.Equal(t, value, *ptr)
			},
		},
		{
			Name: "ZeroValue",
			TestFunc: func(t *testing.T) {
				// Test with zero values for different types
				intPtr := Pointer(0)
				require.NotNil(t, intPtr)
				require.Equal(t, 0, *intPtr)

				strPtr := Pointer("")
				require.NotNil(t, strPtr)
				require.Equal(t, "", *strPtr)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, tc.TestFunc)
	}
}

func TestValue(t *testing.T) {
	cases := []struct {
		Name  string
		Input *int
		Want  int
	}{
		{
			Name:  "NonNil",
			Input: Pointer(42),
			Want:  42,
		},
		{
			Name:  "Nil",
			Input: nil,
			Want:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			have := Value(tc.Input)
			require.Equal(t, tc.Want, have)
		})
	}
}

func TestValueString(t *testing.T) {
	cases := []struct {
		Name  string
		Input *string
		Want  string
	}{
		{
			Name:  "NonNil",
			Input: Pointer("hello"),
			Want:  "hello",
		},
		{
			Name:  "Nil",
			Input: nil,
			Want:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			have := Value(tc.Input)
			require.Equal(t, tc.Want, have)
		})
	}
}

func TestCoalesce(t *testing.T) {
	cases := []struct {
		Name     string
		Fallback int
		Pointers []*int
		Want     int
	}{
		{
			Name:     "FirstNonNil",
			Fallback: 0,
			Pointers: []*int{Pointer(1), Pointer(2), Pointer(3)},
			Want:     1,
		},
		{
			Name:     "SecondNonNil",
			Fallback: 0,
			Pointers: []*int{nil, Pointer(2), Pointer(3)},
			Want:     2,
		},
		{
			Name:     "AllNil",
			Fallback: 99,
			Pointers: []*int{nil, nil, nil},
			Want:     99,
		},
		{
			Name:     "NoPointers",
			Fallback: 42,
			Pointers: []*int{},
			Want:     42,
		},
		{
			Name:     "SingleNonNil",
			Fallback: 0,
			Pointers: []*int{Pointer(5)},
			Want:     5,
		},
		{
			Name:     "SingleNil",
			Fallback: 10,
			Pointers: []*int{nil},
			Want:     10,
		},
		{
			Name:     "MixedWithZeroValue",
			Fallback: -1,
			Pointers: []*int{nil, Pointer(0), Pointer(1)},
			Want:     0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			have := Coalesce(tc.Fallback, tc.Pointers...)
			require.Equal(t, tc.Want, have)
		})
	}
}
