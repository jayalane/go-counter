// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"fmt"
	"log"
	"sync/atomic"
)

var numCalled uint32

// Incr is the main API - will create counter, and add one to it, as needed.
// One line does it all.
func Incr(name string) {
	IncrDelta(name, 1)
}

// IncrSuffix allows you to do an Incr without runtime lookup of the
// caller for the suffix.
func IncrSuffix(name string, suffix string) {
	IncrDeltaSuffix(name, 1, suffix)
}

// IncrSync is the faster API - will create counter, and add one to it, as needed.
// One line does it all.
func IncrSync(name string) {
	IncrDeltaSync(name, 1)
}

// IncrSyncSuffix is the fastest API - will create counter, and add
// one to it, as needed.  One line does it all.
func IncrSyncSuffix(name string, suffix string) {
	IncrDeltaSyncSuffix(name, 1, suffix)
}

// IncrDelta is most versatile API - You can add more than 1 to the counter (negative values are fine).
func IncrDelta(name string, i int64) {
	suffix := getCallerFunctionName()
	IncrDeltaSuffix(name, i, suffix)
}

func getChannel() uint32 {
	newCount := atomic.AddUint32(&numCalled, 1)

	return newCount % numChannels
}

// IncrDeltaSuffix is most versatile API - You can add more than 1 to
// the counter (negative values are fine) and provide a static/fast
// suffix for the counter.
func IncrDeltaSuffix(name string, i int64, suffix string) {
	j := getChannel()

	select {
	case theCtx.c[j] <- counterMsg{name, suffix, i}:
		// good
	default: // bad but ok
	}
}

// ReadSync takes a stat name (including suffix) and returns its value.
func ReadSync(name string) int64 {
	theCtx.ctxLock.RLock()

	c, ok := theCtx.countersByName[name]
	if !ok || c == nil {
		c, ok = theCtx.counters[name]
	}

	defer theCtx.ctxLock.RUnlock()

	if !ok {
		fmt.Println("Can't find", name)

		return 0
	}

	return atomic.LoadInt64(&c.data)
}

// IncrDeltaSync is faster sync more versatile API - You can add more than 1 to the counter (negative values are fine).
func IncrDeltaSync(name string, i int64) {
	suffix := getCallerFunctionName()
	IncrDeltaSyncSuffix(name, i, suffix)
}

// IncrDeltaSyncSuffix is best API.
func IncrDeltaSyncSuffix(name string, i int64, suffix string) {
	getOrMakeAndIncrCounter(name, suffix, i)
}

// Decr is used to decrement a counter made with Incr.
func Decr(name string) {
	IncrDelta(name, -1)
}

// DecrSuffix is used to decrement a counter made with Incr with the
// suffix provided instead of the runtime inspection.
func DecrSuffix(name string, suffix string) {
	IncrDeltaSuffix(name, -1, suffix)
}

func logCounter(name string, mc *counter, data int64) {
	log.Printf(theCtx.fmtString,
		name,
		data,
		data-mc.oldData)
}
