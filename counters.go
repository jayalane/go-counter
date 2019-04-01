// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"log"
	"sort"
	"sync"
	"time"
)

type metaCounter struct {
	name   string
	c1     string
	c2     string
	prefix string
	f      MetaCounterF
}

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
	metaCtrs     map[string]metaCounter
	countersLock sync.RWMutex
	reset        time.Time
	startTime    time.Time
	started      bool
	finished     chan bool
	c            chan counterMsg
	fmtString    string
	fmtStringStr string
	fmtStringF64 string
	timeSleep    float64
}

var theCtx = ctx{}

// InitCounters should be called at least once to start the go routines etc.
func InitCounters() {
	if theCtx.started {
		return
	}
	theCtx.c = make(chan counterMsg, 10000)
	theCtx.finished = make(chan bool, 1)
	theCtx.counters = make(map[string]counter)
	theCtx.metaCtrs = make(map[string]metaCounter)
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
		theCtx.fmtStringF64 = "%-40s  %20f %20f\n"
		if theCtx.timeSleep == 0 {
			theCtx.timeSleep = 60.0
		}
		for {
			n := time.Now()
			log.Printf(theCtx.fmtStringStr, "--------------------------", time.Now(), "")
			log.Printf(theCtx.fmtStringStr, "Uptime", time.Since(theCtx.startTime), "")
			theCtx.countersLock.Lock()
			i := 0
			// do meta counters first before oldData is updated
			mctrNames := make([]string, len(theCtx.counters))
			for k := range theCtx.metaCtrs { // cool scope is only in loop
				mctrNames[i] = theCtx.metaCtrs[k].name
				i++
			}
			log.Printf(theCtx.fmtStringStr, "---M-E-T-A- -C-O-U-N-T----", time.Now(), "")
			sort.Strings(mctrNames)
			for k := range mctrNames {
				logMetaCounter(theCtx.metaCtrs[mctrNames[k]], theCtx.counters)
			}
			ctrNames := make([]string, len(theCtx.counters))
			i = 0
			for k := range theCtx.counters {
				ctrNames[i] = k
				i++
			}
			sort.Strings(ctrNames)
			for k := range ctrNames {
				logCounter(ctrNames[k], theCtx.counters[ctrNames[k]])
				newC := theCtx.counters[ctrNames[k]]
				newC.oldData = newC.data            // have to update old data
				theCtx.counters[ctrNames[k]] = newC // this way
			}
			theCtx.countersLock.Unlock()
			time.Sleep(time.Second * (time.Duration(theCtx.timeSleep) - time.Duration(int64(time.Since(n)/time.Second))))
		}
	}()
}

// Incr is the main API - will create counter, and add one to it, as needed.
// One line does it all.
func Incr(name string) {
	IncrDelta(name, 1)
}

// AddMetaCounter adds in a CB to calculate a new number based on other counters
func AddMetaCounter(name string,
	c1 string,
	c2 string,
	f MetaCounterF) {
	prefix := getCallerFunctionName()
	theCtx.metaCtrs[name] = metaCounter{name, c1, c2, prefix, f}
	log.Println("Meta counters", theCtx.metaCtrs)
}

// MetaCounterF is a function taking two ints and returning a calculated float64 for a new counter-type thing which is derived from 2 other ones
type MetaCounterF func(int64, int64) float64

// IncrDelta is most versatile API - You can add more than 1 to the counter (negative values are fine).
func IncrDelta(name string, i int64) {
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

// RatioTotal can be supplied as a MetaCounter function to calculate e.g. availability between good and bad
func RatioTotal(a int64, b int64) float64 {
	return float64(a) / (float64(a) + float64(b))
}

func logMetaCounter(mc metaCounter, cs map[string]counter) {
	c1, ok := cs[mc.c1]
	if !ok {
		return
	}
	c2, ok := cs[mc.c2]
	if !ok {
		return
	}
	vTotal := mc.f(c1.data, c2.data)
	log.Printf("c1, c1 old, c2, c2 old %d %d %d %d\n",
		c1.data,
		c1.oldData,
		c2.data,
		c2.oldData)
	vDelta := mc.f(c1.data-c1.oldData, c2.data-c2.oldData)

	log.Printf(theCtx.fmtStringF64,
		mc.name+"/"+mc.prefix,
		vTotal,
		vDelta)
}

func logCounter(name string, mc counter) {
	log.Printf(theCtx.fmtString,
		name+"/"+mc.prefix,
		mc.data,
		mc.data-mc.oldData)
}

// SetLogInterval sets the number of seconds to sleep between logs of the counters
func SetLogInterval(i float64) {
	theCtx.timeSleep = i
}

// SetFmtString sets the format string to log the counters with.  It must have a %s and a %d
func SetFmtString(fs string) {
	theCtx.fmtString = fs // should validate
}
