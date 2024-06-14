package runutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackoffTypes(t *testing.T) {
	assert.Implements(t, new(Backoff), ExponentialBackoff{})
	assert.Implements(t, new(Backoff), StaticBackoff{})
}

func TestStaticBackoff(t *testing.T) {
	bo := StaticBackoff{Sleep: 10 * time.Millisecond}
	assert.Equal(t, time.Duration(0), bo.Duration(0))
	for i := 1; i < 10; i++ {
		assert.Equal(t, 10*time.Millisecond, bo.Duration(i))
	}
}

func TestExponentialBackoffWithoutJitter(t *testing.T) {
	cases := []struct {
		bo   ExponentialBackoff
		want []int
	}{
		{
			bo:   ExponentialBackoff{Initial: time.Second, Max: time.Minute},
			want: []int{0, 1, 2, 4, 8, 16, 32, 60, 60, 60, 60},
		},
		{
			bo:   ExponentialBackoff{Initial: 2 * time.Second, Max: time.Minute},
			want: []int{0, 2, 4, 8, 16, 32, 60, 60, 60, 60, 60},
		},
		{
			bo:   ExponentialBackoff{Initial: 3 * time.Second, Max: time.Minute},
			want: []int{0, 3, 6, 12, 24, 48, 60, 60, 60},
		},
	}

	for _, tc := range cases {
		name := fmt.Sprintf("i=%v,m=%v", tc.bo.Initial, tc.bo.Max)
		t.Run(name, func(t *testing.T) {
			require.Equal(t, 0., tc.bo.JitterProportion,
				"jitter contains randomness and cannot be tested here")
			for attempt, expected := range tc.want {
				want := time.Duration(expected) * time.Second
				have := tc.bo.Duration(attempt)
				assert.Equal(t, want, have)
			}
		})
	}
}

func TestExponentialBackoffWithJitter(t *testing.T) {
	cases := []struct {
		bo  ExponentialBackoff
		min []int
		max []int
	}{
		{
			// This is just a sanitiy check for the test itself.
			bo:  ExponentialBackoff{Initial: 2 * time.Second, Max: time.Minute},
			min: []int{0, 2, 4, 8, 16, 32, 60, 60, 60, 60, 60},
			max: []int{0, 2, 4, 8, 16, 32, 60, 60, 60, 60, 60},
		},
		{
			bo:  ExponentialBackoff{Initial: 2 * time.Second, Max: time.Minute, JitterProportion: 0.5},
			min: []int{0, 1, 2, 4, 8, 16, 30, 30, 30, 30, 30},
			max: []int{0, 2, 4, 8, 16, 32, 60, 60, 60, 60, 60},
		},
	}

	for _, tc := range cases {
		name := fmt.Sprintf("i=%v,m=%v,j=%v", tc.bo.Initial, tc.bo.Max, tc.bo.JitterProportion)
		t.Run(name, func(t *testing.T) {
			for attempt := range tc.min {
				wantMin := time.Duration(tc.min[attempt]) * time.Second
				wantMax := time.Duration(tc.max[attempt]) * time.Second
				have := tc.bo.Duration(attempt)

				assert.GreaterOrEqual(t, have, wantMin, "attempt #%d", attempt)
				assert.LessOrEqual(t, have, wantMax, "attempt #%d", attempt)
			}
		})
	}
}

func TestExponentialBackoffWithHighAttempts(t *testing.T) {
	bo := ExponentialBackoff{
		Initial:          time.Minute,
		Max:              5 * time.Minute,
		JitterProportion: 0.5,
	}

	cases := []int{1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8}

	for _, attempt := range cases {
		t.Run(fmt.Sprint(attempt), func(t *testing.T) {
			duration := bo.Duration(attempt)
			assert.Greater(t, duration, time.Duration(0))
		})
	}
}
