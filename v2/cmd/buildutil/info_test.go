package main

import "testing"

func TestParseVersion(t *testing.T) {
	cases := []struct {
		name    string
		in, out string
	}{
		{
			name: "release",
			in:   "v1.6.0",
			out:  "v1.6.0",
		},
		{
			name: "snapshot",
			in:   "v1.6.0-20-gb9e9373",
			out:  "v1.6.0+snapshot.20.b9e9373",
		},
		{
			name: "dirty",
			in:   "v1.6.0-20-gb9e9373-dirty",
			out:  "v1.6.0+dirty.20.b9e9373",
		},
		{
			name: "commit",
			in:   "gb9e9373",
			out:  "v0.0.0+unknown.gb9e9373",
		},
		{
			name: "prerelease",
			in:   "v1.6.0+alpha.1",
			out:  "v1.6.0+alpha.1",
		},
		{
			name: "snapshot-after-prerelease",
			in:   "v1.6.0+alpha.1-20-gb9e9373",
			out:  "v1.6.0+snapshot.20.b9e9373",
		},
		{
			name: "dirty-after-prerelease",
			in:   "v1.6.0+alpha.1-20-gb9e9373-dirty",
			out:  "v1.6.0+dirty.20.b9e9373",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := ParseVersion(tc.in)
			if err != nil {
				t.Fatal(err)
			}

			have := v.String()
			if have != tc.out {
				t.Fatalf(`Have "%s", but want "%s" for "%s".`, have, tc.out, tc.in)
			}
		})
	}

}
