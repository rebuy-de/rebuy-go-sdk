package typeutil

import (
	"fmt"
	"testing"
)

func TestByteFormat(t *testing.T) {
	cases := []struct {
		size int64
		want string
	}{
		{size: 42, want: "42B"},
		{size: 42 * 1024, want: "42.00KiB"},
		{size: 42 * 1024 * 1024, want: "42.00MiB"},
		{size: 42 * 1024 * 1024 * 1024, want: "42.00GiB"},
		{size: 1337, want: "1.306KiB"},
		{size: 1337 * 1024, want: "1.306MiB"},
		{size: 1337 * 1024 * 1024, want: "1.306GiB"},
		{size: 1337 * 1024 * 1024 * 1024, want: "1.306TiB"},
		{size: 1, want: "1B"},
		{size: 11, want: "11B"},
		{size: 111, want: "111B"},
		{size: 1111, want: "1.085KiB"},
		{size: 11111, want: "10.85KiB"},
		{size: 111111, want: "108.5KiB"},
		{size: 1111111, want: "1.060MiB"},
		{size: 11111111, want: "10.60MiB"},
		{size: 111111111, want: "106.0MiB"},
		{size: 1111111111, want: "1.035GiB"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.size), func(t *testing.T) {
			b := JSONBytes{Size: tc.size}
			have := b.String()

			if tc.want != have {
				t.Errorf("%s != %s", tc.want, have)
			}

		})
	}

}
