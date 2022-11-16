// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"fmt"
	"log"
	"math/rand"
	"sync/atomic"
	"time"
)

// Incr is the main API - will create counter, and add one to it, as needed.
// One line does it all.
func Incr(name string) {
	IncrDelta(name, 1)
}

// IncrSuffix allows you to do an Incr without runtime lookup of the
// caller for the suffix
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

// IncrDeltaSuffix is most versatile API - You can add more than 1 to
// the counter (negative values are fine) and provide a static/fast
// suffix for the counter
func IncrDeltaSuffix(name string, i int64, suffix string) {
	j := rand.Uint32() % numChannels
	select {
	case theCtx.c[j] <- counterMsg{name, suffix, i}:
		// good
	default:
		// bad but ok
	}
}

// ReadSync takes a stat name (including suffix) and returns its value
func ReadSync(name string) int64 {
	theCtx.ctxLock.Lock()
	c, ok := theCtx.counters[name]
	theCtx.ctxLock.Unlock()
	if !ok {
		fmt.Println("Can't find", name)
		return 0
	}
	return c.data
}

// IncrDeltaSync is faster sync more versatile API - You can add more than 1 to the counter (negative values are fine).
func IncrDeltaSync(name string, i int64) {
	suffix := getCallerFunctionName()
	IncrDeltaSyncSuffix(name, i, suffix)
}

// IncrDeltaSyncSuffix is best API.
func IncrDeltaSyncSuffix(name string, i int64, suffix string) {

	theCtx.ctxLock.Lock()
	c, ok := theCtx.counters[name+"/"+suffix]
	theCtx.ctxLock.Unlock()
	now := time.Now()
	if !ok {
		c = counter{}
		c.firstSeen = now
	}
	atomic.AddInt64(&c.data, i)
	maxSeenSet := false
	if atomic.LoadInt64(&c.data) > atomic.LoadInt64(&c.maxVal) {
		atomic.StoreInt64(&c.maxVal, c.data)
		maxSeenSet = true
	}
	theCtx.ctxLock.Lock()
	c.lastSeen = now
	if maxSeenSet {
		c.maxSeen = now
	}
	theCtx.counters[name+"/"+suffix] = c
	theCtx.ctxLock.Unlock()
}

// Decr is used to decrement a counter made with Incr.
func Decr(name string) {
	IncrDelta(name, -1)
}

// DecrSuffix is used to decrement a counter made with Incr with the
// suffix provided instead of the runtime inspection
func DecrSuffix(name string, suffix string) {
	IncrDeltaSuffix(name, -1, suffix)
}

func logCounter(name string, mc counter) {
	log.Printf(theCtx.fmtString,
		name,
		mc.data,
		mc.data-mc.oldData)
}
