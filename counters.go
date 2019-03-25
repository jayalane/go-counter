// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"log"
	"time"
)

type counter struct {
	data      int64
	maxVal    int64
	maxSeen   time.Time
	firstSeen time.Time
	lastSeen  time.Time
}

type counterMsg struct {
	name string
	i    int64
}

type ctx struct {
	counters     map[string]counter
	reset        time.Time
	startTime    time.Time
	started      bool
	finished     chan bool
	c            chan counterMsg
	fmtString    string
	fmtStringStr string
	timeSleep    float64
}

var theCtx = ctx{}

// IncrCounter is the main API - will initialize package, create counter, and add one to it, as needed.
// One line does it all.
func IncrCounter(name string) {
	if !theCtx.started {
		startUpRoutine()
	}
	select {
	case theCtx.c <- counterMsg{name, 1}:
		// good
	default:
		// bad but ok
	}
}

// DecrCounter is used to decrement a counter made with IncrCounter
func DecrCounter(name string) {
	if !theCtx.started {
		startUpRoutine()
	}
	select {
	case theCtx.c <- counterMsg{name, -1}:
		// good
	default:
		// bad but ok
	}
}

func startUpRoutine() {
	theCtx.c = make(chan counterMsg, 10000)
	theCtx.finished = make(chan bool, 1)
	theCtx.counters = make(map[string]counter)
	theCtx.started = true
	theCtx.startTime = time.Now()
	go func() { //reader
		for {
			select {
			case <-theCtx.finished:
				return
			case cm := <-theCtx.c:
				str := cm.name
				i := cm.i
				c, ok := theCtx.counters[str]
				n := time.Now()
				if !ok {
					c = counter{}
					c.firstSeen = n
				}
				c.lastSeen = n
				c.data += i // bad name
				if c.data > c.maxVal {
					c.maxVal = c.data
					c.maxSeen = n // same time.Now for all three
				}
				theCtx.counters[str] = c
			default:
				// oh well
			}
		}
	}()

	go func() { // per minute checker
		theCtx.fmtString = "%-40s  %20d\n"
		theCtx.fmtStringStr = "%-40s  %20s\n"
		if theCtx.timeSleep == 0 {
			theCtx.timeSleep = 60.0
		}
		for {
			n := time.Now()
			log.Printf(theCtx.fmtStringStr, "--------------------------", time.Now())
			log.Printf(theCtx.fmtStringStr, "Uptime", time.Since(theCtx.startTime))
			for k, v := range theCtx.counters {
				log.Printf(theCtx.fmtString, k, v.data)
			}
			time.Sleep(time.Second * (time.Duration(theCtx.timeSleep) - time.Duration(int64(time.Since(n)/time.Second))))
		}
	}()
}

// SetLogInterval sets the number of seconds to sleep between logs of the counters
func SetLogInterval(i float64) {
	theCtx.timeSleep = i
}

// SetFmtString sets the format string to log the counters with.  It must have a %s and a %d
func SetFmtString(fs string) {
	theCtx.fmtString = fs // should validate
}
