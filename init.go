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

// ValReport is the minutes change in
// the named metric
type ValReport struct {
	Name  string
	Delta float64
}

// ValReporter is a function callback that can be registered
// to dump metrics once a minute to some other system
type ValReporter func(metrics []ValReport) // callback used below in SetValReporter

type metaCounter struct {
	name string
	c1   string
	c2   string
	f    MetaCounterF
}

type counter struct {
	oldData int64
	data    int64
}

type counterMsg struct {
	name   string
	suffix string
	i      int64
}

type value struct {
	oldData float64
	data    float64
	N       float64
}

type valueMsg struct {
	name   string
	suffix string
	v      float64
}

type ctx struct {
	values       map[string]*value
	counters     map[string]*counter
	metaCtrs     map[string]*metaCounter
	maxLen       int // length of longest metric
	logCb        MetricReporter
	valCb        ValReporter
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
	cbVal := make([]ValReport, len(theCtx.values))       // for CB

	updateMaxLen(&ctrNames, &valNames)
	sort.Strings(valNames)

	for k := range valNames {
		if theCtx.valCb != nil {
			cbVal[k].Name = valNames[k]
			cbVal[k].Delta = theCtx.values[valNames[k]].data - theCtx.values[valNames[k]].oldData
		}
		logValue(valNames[k], theCtx.values[valNames[k]])
		newV := theCtx.values[valNames[k]]
		newV.oldData = newV.data // have to update old data
	}

	sort.Strings(ctrNames)

	for k := range ctrNames {
		data := atomic.LoadInt64(&theCtx.counters[ctrNames[k]].data)
		if theCtx.logCb != nil {
			cbData[k].Name = ctrNames[k]
			cbData[k].Delta = data - theCtx.counters[ctrNames[k]].oldData
		}
		logCounter(ctrNames[k], theCtx.counters[ctrNames[k]], data)
		newC := theCtx.counters[ctrNames[k]]
		newC.oldData = data // have to update old data
	}

	logCb := theCtx.logCb
	valCb := theCtx.valCb

	theCtx.ctxLock.Unlock()

	if logCb != nil {
		logCb(cbData)
	}

	if valCb != nil {
		valCb(cbVal)
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
	theCtx.counters = make(map[string]*counter)
	theCtx.values = make(map[string]*value)
	theCtx.metaCtrs = make(map[string]*metaCounter)
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
					theCtx.ctxLock.RLock()
					c, ok := theCtx.counters[str]
					theCtx.ctxLock.RUnlock()
					if !ok {
						c = &counter{}
						theCtx.ctxLock.Lock()
						c.data = i
						theCtx.counters[str] = c
						theCtx.ctxLock.Unlock()
					} else {
						atomic.AddInt64(&c.data, i) // bad name
					}
					// removed default because this should block
				}
			}
		}(i)
	}

	// values go routines
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
					if !ok {
						v = &value{}
						v.data = val
						theCtx.ctxLock.Lock()
						theCtx.values[str] = v
						theCtx.ctxLock.Unlock()
					} else {
						theCtx.ctxLock.Lock()
						v.data = val // bad but no atomics and just 1/minute
						theCtx.ctxLock.Unlock()
						// removed default because this should block
					}
				}
			}
		}(i)
	}

	go func() { // per minute checker
		theCtx.ctxLock.Lock()
		if theCtx.timeSleep == 0 {
			theCtx.timeSleep = 60.0
		}
		timeSleep := theCtx.timeSleep
		theCtx.ctxLock.Unlock()
		for {
			n := time.Now()
			time.Sleep(time.Second * (time.Duration(timeSleep) - time.Duration(int64(time.Since(n)/time.Second))))
			checkRuntime()
			LogCounters()
		}
	}()
}

// SetMetricReporter specifies a function to be called once per
// LogInterval with the names of the current metrics and the last
// minute delta
func SetMetricReporter(fn MetricReporter) {
	theCtx.ctxLock.Lock()
	theCtx.logCb = fn
	theCtx.ctxLock.Unlock()
}

// SetValReporter specifies a function to be called once per
// LogInterval with the names of the current metrics which are
// float64s and the last minute delta
func SetValReporter(fn ValReporter) {
	theCtx.ctxLock.Lock()
	theCtx.valCb = fn
	theCtx.ctxLock.Unlock()
}

// SetLogInterval sets the number of seconds to sleep between logs of the counters
func SetLogInterval(i float64) {
	theCtx.ctxLock.Lock()
	theCtx.timeSleep = i
	theCtx.ctxLock.Unlock()
}

// SetFmtString sets the format string to log the counters with.  It must have a %s and two %d
func SetFmtString(fs string) {
	theCtx.ctxLock.Lock()
	theCtx.fmtString = fs // should validate
	theCtx.ctxLock.Unlock()
}
