package runutil

import (
	"math"
	"math/rand"
	"time"
)

// Backoff is an interface to calculate the wait times between attemts of doing
// a task. The first attempt must always return 0s. The Duration function
// can be used together with the [Wait] function for a cancelable backoff sleep.
type Backoff interface {
	Duration(int) time.Duration
}

// StaticBackoff always returns the same sleep duration to any but the 0th
// attempt.
type StaticBackoff struct {
	Sleep time.Duration
}

func (b StaticBackoff) Duration(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}
	return b.Sleep
}

// ExponentialBackoff is a typical exponentail backoff with Jitter, based on
// this blog post:
// https://aws.amazon.com/ru/blogs/architecture/exponential-backoff-and-jitter/
type ExponentialBackoff struct {
	Initial          time.Duration
	Max              time.Duration
	JitterProportion float64
}

func (b ExponentialBackoff) Duration(attempt int) time.Duration {
	if attempt == 0 {
		return time.Duration(0)
	}

	var (
		maxWait   = math.Pow(2., float64(attempt-1))
		minWait   = maxWait * (1. - b.JitterProportion)
		jitter    = maxWait * b.JitterProportion * rand.Float64()
		totalWait = minWait + jitter
	)

	// Note: We must do the min() before muliplying with b.Initial, because it
	// is a time.Duration with nano second resolution and we might hit a number
	// overflow quite fast which results in not wait time at all.
	totalWait = min(totalWait, float64(b.Max)/float64(b.Initial))

	return time.Duration(float64(b.Initial) * totalWait)
}
