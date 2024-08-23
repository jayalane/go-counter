// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"log"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// numChannels is the number of API facing channels and reading goroutins
// to reduce lock contention.
const numChannels = 10

// MetricReport is the minutes change in
// the named metric.
type MetricReport struct {
	Name  string
	Delta int64
}

// MetricReporter is a function callback that can be registered
// to dump metrics once a minute to some other system.
type MetricReporter func(metrics []MetricReport) // callback used below in SetMetricReporter

// ValReport is the minutes change in
// the named metric.
type ValReport struct {
	Name  string
	Delta float64
}

// ValReporter is a function callback that can be registered
// to dump metrics once a minute to some other system.
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
	valuesByName   map[string]*value // key present, nil value means check values
	values         map[string]*value
	countersByName map[string]*counter // key present, nil value means check counter
	counters       map[string]*counter
	metaCtrs       map[string]*metaCounter
	maxLen         int // length of longest metric
	logCb          MetricReporter
	valCb          ValReporter
	ctxLock        sync.RWMutex
	startTime      time.Time
	started        bool
	finished       chan bool
	c              []chan counterMsg
	v              []chan valueMsg
	fmtString      string
	fmtStringStr   string
	fmtStringF64   string
	timeSleep      float64
}

var theCtx = ctx{}

// LogCounters prints out the counters.  It is called internally
// each minute but can be called externally e.g. at process end.
func LogCounters() {
	theCtx.ctxLock.Lock()
	updateMaxLen(nil, nil)

	theCtx.fmtString = "%-" + strconv.Itoa(theCtx.maxLen+12) + "s  %20d %20d\n"    //nolint:mnd
	theCtx.fmtStringStr = "%-" + strconv.Itoa(theCtx.maxLen+12) + "s  %20s %20s\n" //nolint:mnd
	theCtx.fmtStringF64 = "%-" + strconv.Itoa(theCtx.maxLen+12) + "s  %20f %20f\n" //nolint:mnd
	fmtStringStr := theCtx.fmtStringStr

	theCtx.ctxLock.Unlock()

	log.Printf(fmtStringStr, "--------------------------", time.Now(), "")
	log.Printf(fmtStringStr, "Uptime", time.Since(theCtx.startTime), "")

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
	ctrNames := make([]string, len(theCtx.counters)+len(theCtx.countersByName))
	valNames := make([]string, len(theCtx.values)+len(theCtx.valuesByName))
	cbData := make([]MetricReport, len(theCtx.counters)+len(theCtx.countersByName)) // for CB
	cbVal := make([]ValReport, len(theCtx.values)+len(theCtx.valuesByName))         // for CB

	updateMaxLen(&ctrNames, &valNames)
	sort.Strings(valNames)

	for k := range valNames {
		v, ok := theCtx.valuesByName[valNames[k]]
		if !ok || v == nil {
			v = theCtx.values[valNames[k]]
		}
		if theCtx.valCb != nil {
			cbVal[k].Name = valNames[k]
			cbVal[k].Delta = v.data - v.oldData
		}

		logValue(valNames[k], v)

		newV := v
		newV.oldData = newV.data // have to update old data
	}

	sort.Strings(ctrNames)

	for k := range ctrNames {
		v, ok := theCtx.countersByName[ctrNames[k]]
		if !ok || v == nil {
			v = theCtx.counters[ctrNames[k]]
		}
		data := atomic.LoadInt64(&v.data)

		if theCtx.logCb != nil {
			cbData[k].Name = ctrNames[k]
			cbData[k].Delta = data - v.oldData
		}

		logCounter(ctrNames[k], v, data)

		newC := v
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

// updateMaxLen updates the max len for formatting for both vals and ctrs.
func updateMaxLen(ctrNames *[]string, valNames *[]string) {
	maxLen := 0

	i := 0
	for k := range theCtx.counters {
		if len(k) > maxLen {
			maxLen = len(k)
		}

		if ctrNames != nil {
			(*ctrNames)[i] = k
		}

		i++
	}
	for k := range theCtx.countersByName {
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
	for k := range theCtx.valuesByName {
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

func initCtx() {
	theCtx.c = make([]chan counterMsg, numChannels)

	for i := range numChannels {
		theCtx.c[i] = make(chan counterMsg, 100000) //nolint:mnd
	}

	theCtx.v = make([]chan valueMsg, numChannels)

	for i := range numChannels {
		theCtx.v[i] = make(chan valueMsg, 100000) //nolint:mnd
	}

	theCtx.finished = make(chan bool, 1)
	theCtx.counters = make(map[string]*counter)
	theCtx.countersByName = make(map[string]*counter)
	theCtx.values = make(map[string]*value)
	theCtx.valuesByName = make(map[string]*value)
	theCtx.metaCtrs = make(map[string]*metaCounter)
	theCtx.started = true
	theCtx.startTime = time.Now()
}

func minuteGoRoutine() {
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
}

func readingValGoRoutine(index int) {
	for {
		select {
		// no default because this should block
		case <-theCtx.finished:
			return
		case vm := <-theCtx.v[index]:
			getOrMakeAndSetValue(vm.name, vm.suffix, vm.v)
		}
	}
}

// getOrMakeCounter checks the 2 hashes and increments the appropriate place.
func getOrMakeAndIncrCounter(name string, suffix string, i int64) {
	nameOnly := true // true means its in countersByName
	key := ""

	theCtx.ctxLock.RLock()

	c, ok := theCtx.countersByName[name]

	theCtx.ctxLock.RUnlock()

	if ok && c == nil {
		theCtx.ctxLock.RLock()

		fullName := name + "/" + suffix
		nameOnly = false
		key = fullName
		c, ok = theCtx.countersByName[fullName]

		theCtx.ctxLock.RUnlock()
	} else {
		key = name
	}

	if !ok {
		c = &counter{}
		c.data = i

		theCtx.ctxLock.Lock()

		if nameOnly {
			theCtx.countersByName[key] = c
		} else {
			theCtx.counters[key] = c
		}

		theCtx.ctxLock.Unlock()
	} else {
		atomic.AddInt64(&c.data, i)
	}
}

// getOrMakeValue returns a counter struct.
func getOrMakeAndSetValue(name string, suffix string, v float64) {
	nameOnly := true
	key := ""

	theCtx.ctxLock.RLock()

	c, ok := theCtx.valuesByName[name]

	theCtx.ctxLock.RUnlock()

	if ok && c == nil {
		theCtx.ctxLock.RLock()
		nameOnly = false
		fullName := name + "/" + suffix // will malloc
		key = fullName
		c, ok = theCtx.valuesByName[fullName]
		theCtx.ctxLock.RUnlock()
	} else {
		key = name
	}

	if !ok {
		c = &value{}
		c.data = v

		theCtx.ctxLock.Lock()
		if nameOnly {
			theCtx.valuesByName[key] = c
		} else {
			theCtx.values[key] = c
		}

		theCtx.ctxLock.Unlock()
	} else {
		theCtx.ctxLock.Lock()

		c.data = v // bad but no atomics and just 1/minute (from go stats)

		theCtx.ctxLock.Unlock()
	}
}

func readingCountsGoRoutine(index int) {
	for {
		select {
		// no default because this should block
		case <-theCtx.finished:
			return
		case cm := <-theCtx.c[index]:
			getOrMakeAndIncrCounter(cm.name, cm.suffix, cm.i)
		}
	}
}

// InitCounters should be called at least once to start the go routines etc.
func InitCounters() {
	theCtx.ctxLock.Lock()
	defer theCtx.ctxLock.Unlock()

	if theCtx.started {
		return
	}

	initCtx()

	// counters go routines
	for i := range numChannels {
		go readingCountsGoRoutine(i)
	}

	// values go routines
	for i := range numChannels {
		go readingValGoRoutine(i)
	}

	go minuteGoRoutine()
}

// SetMetricReporter specifies a function to be called once per
// LogInterval with the names of the current metrics and the last
// minute delta.
func SetMetricReporter(fn MetricReporter) {
	theCtx.ctxLock.Lock()
	theCtx.logCb = fn
	theCtx.ctxLock.Unlock()
}

// SetValReporter specifies a function to be called once per
// LogInterval with the names of the current metrics which are
// float64s and the last minute delta.
func SetValReporter(fn ValReporter) {
	theCtx.ctxLock.Lock()
	theCtx.valCb = fn
	theCtx.ctxLock.Unlock()
}

// SetLogInterval sets the number of seconds to sleep between logs of the counters.
func SetLogInterval(i float64) {
	theCtx.ctxLock.Lock()
	theCtx.timeSleep = i
	theCtx.ctxLock.Unlock()
}

// SetFmtString sets the format string to log the counters with.  It must have a %s and two %d.
func SetFmtString(fs string) {
	theCtx.ctxLock.Lock()
	theCtx.fmtString = fs // should validate
	theCtx.ctxLock.Unlock()
}
