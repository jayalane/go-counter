// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"log"
	"sort"
	"sync"
	"time"
)

// MetaCounterF is a function taking two ints and returning a calculated float64 for a new counter-type thing which is derived from 2 other ones
type MetaCounterF func(int64, int64) float64

type metaCounter struct {
	name   string
	c1     string
	c2     string
	prefix string
	oldV   float64
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

// Incr is the main API - will initialize package, create counter, and add one to it, as needed.
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
	theCtx.metaCtrs[name] = metaCounter{name, c1, c2, prefix, 0.0, f}
	log.Println("Meta counters", theCtx.metaCtrs)
}

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

func logMetaCounter(mc metaCounter, cs map[string]counter) metaCounter {
	newMc := mc
	var v float64
	c1, ok := cs[mc.c1]
	if !ok {
		return mc
	}
	c2, ok := cs[mc.c2]
	if !ok {
		return mc
	}
	v = mc.f(c1.data, c2.data)
	log.Printf(theCtx.fmtStringF64,
		mc.name+"/"+mc.prefix,
		v,
		v-mc.oldV)
	newMc.oldV = v
	return newMc
}

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
			m := make([]string, len(theCtx.counters)+len(theCtx.metaCtrs))
			i := 0
			for k := range theCtx.counters {
				m[i] = k
				i++
			}
			for k := range theCtx.metaCtrs {
				m[i] = theCtx.metaCtrs[k].name
				i++
			}
			sort.Strings(m)
			for k := range m {
				_, ok := theCtx.counters[m[k]]
				if ok {
					log.Printf(theCtx.fmtString,
						m[k]+"/"+theCtx.counters[m[k]].prefix,
						theCtx.counters[m[k]].data,
						theCtx.counters[m[k]].data-theCtx.counters[m[k]].oldData)
					newC := theCtx.counters[m[k]]
					newC.oldData = newC.data
					theCtx.counters[m[k]] = newC
				} else {
					theCtx.metaCtrs[m[k]] = logMetaCounter(theCtx.metaCtrs[m[k]], theCtx.counters)
				}
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
