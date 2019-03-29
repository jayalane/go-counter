// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"testing"
	"time"
)

func TestCounter(t *testing.T) {

	InitCounters()
	SetLogInterval(1)
	AddMetaCounter("availability", "good", "bad", RatioTotal)
	Incr("num_of_things")
	Incr("a_num_of_things")
	IncrDelta("good", 97)
	IncrDelta("bad", 3)
	time.Sleep(1100 * time.Millisecond)
	Decr("num_of_things")
	time.Sleep(1100 * time.Millisecond)

}
