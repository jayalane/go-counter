// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
// implemented using channels now but switching to sync based on these tests maybe; will keep both implementations
package counters

import (
	"fmt"
	"sync/atomic"
	"testing"
)

var cbRan int32

func metricReporterCB(metrics []MetricReport) {
	fmt.Println("CB called", len(metrics))

	someThingNotOne := false

	for x := range metrics {
		m := metrics[x]
		val := ReadSync(m.Name)

		if val != 0 && val != 1 {
			someThingNotOne = true
		}
	}

	if !someThingNotOne {
		fmt.Println("Nothing was set")
	} else {
		atomic.StoreInt32(&cbRan, 1)
	}
}

var valCbRan int32

func valReporterCB(metrics []ValReport) {
	atomic.StoreInt32(&valCbRan, 1)

	for x := range metrics {
		m := metrics[x]
		fmt.Println("Val CB: ", m.Name, m.Delta)
	}
}

func BenchmarkCounter(b *testing.B) {
	InitCounters()
	SetLogInterval(1)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Incr("num_of_things")
	}
}

func BenchmarkSyncIncr(b *testing.B) {
	InitCounters()
	SetLogInterval(1)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		IncrSync("num_of_things")
	}
}

func TestCounter(t *testing.T) {
	InitCounters()
	SetLogInterval(10)
	SetMetricReporter(metricReporterCB)
	SetValReporter(valReporterCB)
	Set("floater", 3.141)
	LogCounters()
	AddMetaCounter("availability", "good", "bad", RatioTotal)

	for range 1000 {
		go func() {
			Incr("num_of_things")
			IncrSync("a_num_of_things")
		}()
	}

	IncrDelta("good", 97)
	IncrDeltaSync("bad", 3)
	Set("floater", 3.141)

	for range 20 {
		Decr("num_of_things_2")
	}

	c := atomic.LoadInt32(&cbRan)
	if c != 1 {
		fmt.Println("Callback did not run", c)
		t.Fail()
	}

	cc := atomic.LoadInt32(&valCbRan)
	if cc != 1 {
		fmt.Println("Val callback did not run", cc)
		t.Fail()
		panic("Val cb")
	}

	LogCounters()
}
