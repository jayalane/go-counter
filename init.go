// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// numChannels is the number of API facing channels and reading goroutins
// to reduce lock contention
const numChannels = 10

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
	name string
	c1   string
	c2   string
	f    MetaCounterF
}

type counter struct {
	oldData   int64
	data      int64
	maxVal    int64
	maxSeen   time.Time
	firstSeen time.Time
	lastSeen  time.Time
}

type counterMsg struct {
	name   string
	suffix string
	i      int64
}

type value struct {
	oldData   float64
	data      float64
	maxVal    float64
	minVal    float64
	N         float64
	maxSeen   time.Time
	minSeen   time.Time
	firstSeen time.Time
	lastSeen  time.Time
}

type valueMsg struct {
	name   string
	suffix string
	v      float64
}

type ctx struct {
	values       map[string]value
	counters     map[string]counter
	metaCtrs     map[string]metaCounter
	maxLen       int // length of longest metric
	logCb        MetricReporter
	ctxLock      sync.RWMutex
	startTime    time.Time
	started      bool
	finished     chan bool
	c            []chan counterMsg
	v            []chan valueMsg
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

	theCtx.ctxLock.RLock()
	updateMaxLen(nil, nil)
	theCtx.fmtString = "%-" + fmt.Sprintf("%d", theCtx.maxLen+12) + "s  %20d %20d\n"
	theCtx.fmtStringStr = "%-" + fmt.Sprintf("%d", theCtx.maxLen+12) + "s  %20s %20s\n"
	theCtx.fmtStringF64 = "%-" + fmt.Sprintf("%d", theCtx.maxLen+12) + "s  %20f %20f\n"
	theCtx.ctxLock.RUnlock()

	log.Printf(theCtx.fmtStringStr, "--------------------------", time.Now(), "")
	log.Printf(theCtx.fmtStringStr, "Uptime", time.Since(theCtx.startTime), "")
	theCtx.ctxLock.Lock()
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
	// then the counters
	ctrNames := make([]string, len(theCtx.counters))
	valNames := make([]string, len(theCtx.values))
	cbData := make([]MetricReport, len(theCtx.counters)) // for CB
	updateMaxLen(&ctrNames, &valNames)
	sort.Strings(ctrNames)
	sort.Strings(valNames)
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
	theCtx.ctxLock.Unlock()

	// the the values
	theCtx.ctxLock.Lock()
	for k := range valNames {
		// TODO
		logValue(valNames[k], theCtx.values[valNames[k]])
		newV := theCtx.values[valNames[k]]
		newV.oldData = newV.data          // have to update old data
		theCtx.values[valNames[k]] = newV // this way
	}
	theCtx.ctxLock.Unlock()
	if theCtx.logCb != nil {
		theCtx.logCb(cbData)
	}
}

// maxLen updates the max len for formatting for both vals and ctrs
func updateMaxLen(ctrNames *[]string, valNames *[]string) {
	i := 0
	maxLen := 0
	for k := range theCtx.counters {
		if len(k) > maxLen {
			maxLen = len(k)
		}
		if ctrNames != nil {
			(*ctrNames)[i] = k
		}
		i++
	}
	i = 0
	for k := range theCtx.values {
		if len(k) > maxLen {
			maxLen = len(k)
		}
		if valNames != nil {
			(*valNames)[i] = k
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
	theCtx.c = make([]chan counterMsg, numChannels)
	for i := 0; i < numChannels; i++ {
		theCtx.c[i] = make(chan counterMsg, 100000)
	}
	theCtx.v = make([]chan valueMsg, numChannels)
	for i := 0; i < numChannels; i++ {
		theCtx.v[i] = make(chan valueMsg, 100000)
	}
	theCtx.finished = make(chan bool, 1)
	theCtx.counters = make(map[string]counter)
	theCtx.values = make(map[string]value)
	theCtx.metaCtrs = make(map[string]metaCounter)
	theCtx.started = true
	theCtx.startTime = time.Now()
	// counters go routines
	theCtx.ctxLock = sync.RWMutex{}
	for i := 0; i < numChannels; i++ {
		go func(index int) { //reader
			for {
				select {
				case <-theCtx.finished:
					return
				case cm := <-theCtx.c[index]:
					str := cm.name + "/" + cm.suffix
					i := cm.i
					theCtx.ctxLock.Lock()
					c, ok := theCtx.counters[str]
					theCtx.ctxLock.Unlock()
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
					theCtx.ctxLock.Lock()
					theCtx.counters[str] = c // I forget why this is needed.
					theCtx.ctxLock.Unlock()
					// removed default because this should block
				}
			}
		}(i)
	}

	// values go routines
	theCtx.ctxLock = sync.RWMutex{}
	for i := 0; i < numChannels; i++ {
		go func(index int) { //reader
			for {
				select {
				case <-theCtx.finished:
					return
				case vm := <-theCtx.v[index]:
					str := vm.name + "/" + vm.suffix
					val := vm.v
					theCtx.ctxLock.Lock()
					v, ok := theCtx.values[str]
					theCtx.ctxLock.Unlock()
					now := time.Now()
					if !ok {
						v = value{}
						v.firstSeen = now
					}
					v.lastSeen = now
					v.data = val
					if v.data > v.maxVal {
						v.maxVal = v.data
						v.maxSeen = now
					}
					if v.data < v.minVal {
						v.minVal = v.data
						v.minSeen = now
					}
					theCtx.ctxLock.Lock()
					theCtx.values[str] = v // I forget why this is needed.
					theCtx.ctxLock.Unlock()
					// removed default because this should block
				}
			}
		}(i)
	}

	go func() { // per minute checker
		theCtxLock.Lock()
		if theCtx.timeSleep == 0 {
			theCtx.timeSleep = 60.0
		}
		timeSleep := theCtx.timeSleep
		theCtxLock.Unlock()
		for {
			checkRuntime()
			n := time.Now()
			time.Sleep(time.Second * (time.Duration(timeSleep) - time.Duration(int64(time.Since(n)/time.Second))))
			LogCounters()
		}
	}()
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
