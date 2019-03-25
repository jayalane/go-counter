// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	SetLogInterval(1)
	Incr("num_of_things")
	time.Sleep(1100 * time.Millisecond)
	Decr("num_of_things")
	time.Sleep(1100 * time.Millisecond)
}
