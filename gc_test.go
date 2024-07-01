// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
// implemented using channels now but switching to sync based on these tests maybe; will keep both implementations
package counters

import (
	"testing"
	"time"
)

func TestGcMetrics(_ *testing.T) {
	checkRuntime()
	time.Sleep(2 * time.Second)
	LogCounters()
}
