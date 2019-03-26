// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"log"
	"sort"
	"sync"
	"time"
)

type counter struct {
	oldData   int64
	data      int64
	maxVal    int64
	maxSeen   time.Time
	firstSeen time.Time
	lastSeen  time.Time
	prefix    string
}

type counterMsg struct {
	name   string
	prefix string
	i      int64
}

type ctx struct {
	counters     map[string]counter
	countersLock sync.RWMutex
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

// Incr is the main API - will initialize package, create counter, and add one to it, as needed.
// One line does it all.
func Incr(name string) {
	IncrDelta(name, 1)
}

// IncrDelta is most versatile API - You can add more than 1 to the counter (negative values are fine).
func IncrDelta(name string, i int64) {
	if !theCtx.started {
		startUpRoutine()
	}
	prefix := getCallerFunctionName()
	select {
	case theCtx.c <- counterMsg{name, prefix, i}:
		// good
	default:
		// bad but ok
	}
}

// Decr is used to decrement a counter made with Incr.
func Decr(name string) {
	IncrDelta(name, -1)
}

func startUpRoutine() {
	theCtx.c = make(chan counterMsg, 10000)
	theCtx.finished = make(chan bool, 1)
	theCtx.counters = make(map[string]counter)
	theCtx.started = true
	theCtx.startTime = time.Now()
	theCtx.countersLock = sync.RWMutex{}
	go func() { //reader
		for {
			select {
			case <-theCtx.finished:
				return
			case cm := <-theCtx.c:
				str := cm.name
				prefix := cm.prefix
				i := cm.i
				theCtx.countersLock.Lock()
				c, ok := theCtx.counters[str]
				theCtx.countersLock.Unlock()
				n := time.Now()
				if !ok {
					c = counter{}
					c.firstSeen = n
					c.prefix = prefix
				}
				c.lastSeen = n
				c.data += i // bad name
				if c.data > c.maxVal {
					c.maxVal = c.data
					c.maxSeen = n // same time.Now for all three
				}
				theCtx.countersLock.Lock()
				theCtx.counters[str] = c
				theCtx.countersLock.Unlock()

			default:
				// oh well
			}
		}
	}()

	go func() { // per minute checker
		theCtx.fmtString = "%-40s  %20d %20d\n"
		theCtx.fmtStringStr = "%-40s  %20s %20s\n"
		if theCtx.timeSleep == 0 {
			theCtx.timeSleep = 60.0
		}
		for {
			n := time.Now()
			log.Printf(theCtx.fmtStringStr, "--------------------------", time.Now(), "")
			log.Printf(theCtx.fmtStringStr, "Uptime", time.Since(theCtx.startTime), "")
			theCtx.countersLock.Lock()
			m := make([]string, len(theCtx.counters))
			i := 0
			for k := range theCtx.counters {
				m[i] = k
				i++
			}
			sort.Strings(m)
			for k := range m {
				log.Printf(theCtx.fmtString,
					theCtx.counters[m[k]].prefix+"-"+m[k],
					theCtx.counters[m[k]].data,
					theCtx.counters[m[k]].data-theCtx.counters[m[k]].oldData)
				newC := theCtx.counters[m[k]]
				newC.oldData = newC.data
				theCtx.counters[m[k]] = newC
			}
			theCtx.countersLock.Unlock()
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
