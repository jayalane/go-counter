// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// MetricReport is the minutes change in
// the named named
type MetricReport struct {
	Name  string
	Delta int64
}

// MetricReporter is a function callback that can be registered
// to dump metrics once a minute to some other system
type MetricReporter func(metrics []MetricReport) // callback used below in SetMetricReporter

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
	logCb        MetricReporter
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
var theCtxLock = sync.RWMutex{}

// LogCounters prints out the counters.  It is called internally
// each minute but can be called externally e.g. at process end.
func LogCounters() {

	log.Printf(theCtx.fmtStringStr, "--------------------------", time.Now(), "")
	log.Printf(theCtx.fmtStringStr, "Uptime", time.Since(theCtx.startTime), "")
	theCtx.countersLock.Lock()
	i := 0
	// do meta counters first before oldData is updated
	mctrNames := make([]string, len(theCtx.metaCtrs))
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
	cbData := make([]MetricReport, len(theCtx.counters)) // for CB
	i = 0
	for k := range theCtx.counters {
		ctrNames[i] = k
		i++
	}
	sort.Strings(ctrNames)
	for k := range ctrNames {
		if theCtx.logCb != nil {
			cbData[k].Name = ctrNames[k]
			cbData[k].Delta = theCtx.counters[ctrNames[k]].data - theCtx.counters[ctrNames[k]].oldData
		}
		logCounter(ctrNames[k], theCtx.counters[ctrNames[k]])
		newC := theCtx.counters[ctrNames[k]]
		newC.oldData = newC.data            // have to update old data
		theCtx.counters[ctrNames[k]] = newC // this way
	}
	theCtx.countersLock.Unlock()
	if theCtx.logCb != nil {
		theCtx.logCb(cbData)
	}
}

// InitCounters should be called at least once to start the go routines etc.
func InitCounters() {
	theCtxLock.Lock()
	defer theCtxLock.Unlock()
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
				// removed default because this should block
			}
		}
	}()

	go func() { // per minute checker
		theCtxLock.Lock()
		theCtx.fmtString = "%-90s  %20d %20d\n"
		theCtx.fmtStringStr = "%-90s  %20s %20s\n"
		theCtx.fmtStringF64 = "%-90s  %20f %20f\n"
		if theCtx.timeSleep == 0 {
			theCtx.timeSleep = 60.0
		}
		theCtxLock.Unlock()
		for {
			n := time.Now()
			time.Sleep(time.Second * (time.Duration(theCtx.timeSleep) - time.Duration(int64(time.Since(n)/time.Second))))
			LogCounters()
		}
	}()
}

// Incr is the main API - will create counter, and add one to it, as needed.
// One line does it all.
func Incr(name string) {
	IncrDelta(name, 1)
}

// IncrSync is the faster API - will create counter, and add one to it, as needed.
// One line does it all.
func IncrSync(name string) {
	IncrDelta(name, 1)
}

// AddMetaCounter adds in a CB to calculate a new number based on other counters
func AddMetaCounter(name string,
	c1 string,
	c2 string,
	f MetaCounterF) {
	prefix := getCallerFunctionName()
	theCtxLock.Lock()
	theCtx.metaCtrs[name] = metaCounter{name, c1, c2, prefix, f}
	theCtxLock.Unlock()
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

// ReadSync takes a stat name and returns its value
func ReadSync(name string) int64 {
	theCtx.countersLock.Lock()
	c, ok := theCtx.counters[name]
	theCtx.countersLock.Unlock()
	if !ok {
		fmt.Println("Can't find", name)
		return 0
	}
	// skip the prefix check - name is unique anyways.
	return c.data
}

// IncrDeltaSync is faster sync more versatile API - You can add more than 1 to the counter (negative values are fine).
func IncrDeltaSync(name string, i int64) {
	prefix := getCallerFunctionName()
	theCtx.countersLock.Lock()
	c, ok := theCtx.counters[name]
	theCtx.countersLock.Unlock()
	n := time.Now()
	if !ok {
		c = counter{}
		c.firstSeen = n
		c.prefix = prefix
	}
	atomic.AddInt64(&c.data, i)
	maxSeenSet := false
	if atomic.LoadInt64(&c.data) > atomic.LoadInt64(&c.maxVal) {
		atomic.StoreInt64(&c.maxVal, c.data)
		maxSeenSet = true
	}
	theCtx.countersLock.Lock()
	c.lastSeen = n
	if maxSeenSet {
		c.maxSeen = n
	}
	theCtx.counters[name] = c
	theCtx.countersLock.Unlock()
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

// SetMetricReporter specifies a function to be called once per
// LogInterval with the names of the current metrics and the last
// minute delta
func SetMetricReporter(fn MetricReporter) {
	theCtxLock.Lock()
	theCtx.logCb = fn
	theCtxLock.Unlock()
}

// SetLogInterval sets the number of seconds to sleep between logs of the counters
func SetLogInterval(i float64) {
	theCtxLock.Lock()
	theCtx.timeSleep = i
	theCtxLock.Unlock()
}

// SetFmtString sets the format string to log the counters with.  It must have a %s and a %d
func SetFmtString(fs string) {
	theCtxLock.Lock()
	theCtx.fmtString = fs // should validate
	theCtxLock.Unlock()
}
