// -*- tab-width: 2 -*-

package counters

import (
	"time"
)

// the API in this file allows a func() { } to be timed

// TimeFunc is just a function with no params or return value.
type TimeFunc func()

// TimeFuncRun runs the function and then
// marks it in a histogram.
func TimeFuncRun(name string, f TimeFunc) {
	start := time.Now()

	f()

	end := time.Now()

	MarkDistribution(name,
		end.Sub(start).Seconds())
}

// TimeFuncRunSuffix runs the function and then
// marks it in a histogram.
func TimeFuncRunSuffix(name string, f TimeFunc, suffix string) {
	start := time.Now()

	f()

	end := time.Now()

	MarkDistributionSuffix(name,
		end.Sub(start).Seconds(),
		suffix)
}
