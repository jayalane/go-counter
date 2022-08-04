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
// the named metric
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
	suffix string
	f      MetaCounterF
}

type counter struct {
	oldData   int64
	data      int64
	maxVal    int64
	maxSeen   time.Time
	firstSeen time.Time
	lastSeen  time.Time
	suffix    string
}

type counterMsg struct {
	name   string
	suffix string
	i      int64
}

type ctx struct {
	counters     map[string]counter
	metaCtrs     map[string]metaCounter
	maxLen       int // length of longest metric
	logCb        MetricReporter
	countersLock sync.RWMutex
	// reset        time.Time // TODO
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

	theCtx.countersLock.RLock()
	updateMaxLen(nil)
	theCtx.fmtString = "%-" + fmt.Sprintf("%d", theCtx.maxLen+12) + "s  %20d %20d\n"
	theCtx.fmtStringStr = "%-" + fmt.Sprintf("%d", theCtx.maxLen+12) + "s  %20s %20s\n"
	theCtx.fmtStringF64 = "%-" + fmt.Sprintf("%d", theCtx.maxLen+12) + "s  %20f %20f\n"
	theCtx.countersLock.RUnlock()

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
	updateMaxLen(&ctrNames)
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

// maxLen updates the max len for formatting
func updateMaxLen(ctrNames *[]string) {
	i := 0
	maxLen := 0
	for k := range theCtx.counters {
		if len(k)+len(theCtx.counters[k].suffix) > maxLen {
			maxLen = len(k) + len(theCtx.counters[k].suffix)
		}
		if ctrNames != nil {
			(*ctrNames)[i] = k
		}
		i++
	}
	theCtx.maxLen = maxLen
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
				suffix := cm.suffix
				i := cm.i
				theCtx.countersLock.Lock()
				c, ok := theCtx.counters[str]
				theCtx.countersLock.Unlock()
				n := time.Now()
				if !ok {
					c = counter{}
					c.firstSeen = n
					c.suffix = suffix
				}
				c.lastSeen = n
				c.data += i // bad name
				if c.data > c.maxVal {
					c.maxVal = c.data
					c.maxSeen = n // same time.Now for all three
				}
				theCtx.countersLock.Lock()
				theCtx.counters[str] = c // I forget why this is needed.
				theCtx.countersLock.Unlock()
				// removed default because this should block
			}
		}
	}()

	go func() { // per minute checker
		theCtxLock.Lock()
		if theCtx.timeSleep == 0 {
			theCtx.timeSleep = 60.0
		}
		timeSleep := theCtx.timeSleep
		theCtxLock.Unlock()
		for {
			n := time.Now()
			time.Sleep(time.Second * (time.Duration(timeSleep) - time.Duration(int64(time.Since(n)/time.Second))))
			LogCounters()
		}
	}()
}

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

// AddMetaCounter adds in a CB to calculate a new number based on other counters
func AddMetaCounter(name string,
	c1 string,
	c2 string,
	f MetaCounterF) {
	suffix := getCallerFunctionName()
	theCtxLock.Lock()
	theCtx.metaCtrs[name] = metaCounter{name, c1, c2, suffix, f}
	theCtxLock.Unlock()
}

// MetaCounterF is a function taking two ints and returning a calculated float64 for a new counter-type thing which is derived from 2 other ones
type MetaCounterF func(int64, int64) float64

// IncrDelta is most versatile API - You can add more than 1 to the counter (negative values are fine).
func IncrDelta(name string, i int64) {
	suffix := getCallerFunctionName()
	select {
	case theCtx.c <- counterMsg{name, suffix, i}:
		// good
	default:
		// bad but ok
	}
}

// IncrDeltaSuffix is most versatile API - You can add more than 1 to
// the counter (negative values are fine) and provide a static/fast
// suffix for the counter
func IncrDeltaSuffix(name string, i int64, suffix string) {

	select {
	case theCtx.c <- counterMsg{name, suffix, i}:
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
	// skip the suffix check - name is unique anyways.
	return c.data
}

// IncrDeltaSync is faster sync more versatile API - You can add more than 1 to the counter (negative values are fine).
func IncrDeltaSync(name string, i int64) {
	suffix := getCallerFunctionName()
	IncrDeltaSyncSuffix(name, i, suffix)
}

// IncrDeltaSyncSuffix is best API.
func IncrDeltaSyncSuffix(name string, i int64, suffix string) {

	theCtx.countersLock.Lock()
	c, ok := theCtx.counters[name]
	theCtx.countersLock.Unlock()
	n := time.Now()
	if !ok {
		c = counter{}
		c.firstSeen = n
		c.suffix = suffix
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

// DecrSuffix is used to decrement a counter made with Incr with the
// suffix provided instead of the runtime inspection
func DecrSuffix(name string, suffix string) {
	IncrDeltaSuffix(name, -1, suffix)
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
		mc.name+"/"+mc.suffix,
		vTotal,
		vDelta)
}

func logCounter(name string, mc counter) {
	log.Printf(theCtx.fmtString,
		name+"/"+mc.suffix,
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

// SetFmtString sets the format string to log the counters with.  It must have a %s and two %d
func SetFmtString(fs string) {
	theCtxLock.Lock()
	theCtx.fmtString = fs // should validate
	theCtxLock.Unlock()
}
