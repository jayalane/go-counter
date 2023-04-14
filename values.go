// -*- tab-width: 2 -*-

package counters

// this file has implementations for "value" type metrics (e.g. CPU usage, # go routines

import (
	"log"
	"math/rand"
	"strings"
)

// Set is the main value API - will create value metric, and get the
// caller func for suffix, as needed.  One line does it all.
func Set(name string, val float64) {
	suffix := getCallerFunctionName()
	SetSuffix(name, val, suffix)
}

// SetSuffix is a bit faster API - the func name lookup is a bit slow
func SetSuffix(name string, val float64, suffix string) {
	j := rand.Uint32() % numChannels // for less contention
	select {                         // non-blocking will drop overflow
	case theCtx.v[j] <- valueMsg{name, suffix, val}:
		// good
	default:
		// bad but ok
	}
}

func logValue(name string, mc *value) {
	fmtString := strings.ReplaceAll(theCtx.fmtString, "d", "f") // fragile
	log.Printf(fmtString,
		name,
		mc.data,
		mc.data-mc.oldData)
}
