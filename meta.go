// -*- tab-width: 2 -*-

// Package counters enables 1 line creation of stats to track your program flow; you get summaries every minute
package counters

import (
	"log"
)

// AddMetaCounter adds in a CB to calculate a new number based on other counters.
func AddMetaCounter(name string,
	c1 string,
	c2 string,
	f MetaCounterF,
) {
	suffix := getCallerFunctionName()

	theCtx.ctxLock.Lock()

	// adding the suffix to counter names keeps APi compatibility but is less useful
	theCtx.metaCtrs[name+"/"+suffix] = &metaCounter{name + "/" + suffix, c1 + "/" + suffix, c2 + "/" + suffix, f}

	theCtx.ctxLock.Unlock()
}

// MetaCounterF is a function taking two ints and returning a calculated float64 for a new counter-type thing which is derived from 2 other ones.
type MetaCounterF func(int64, int64) float64

// RatioTotal can be supplied as a MetaCounter function to calculate e.g. availability between good and bad.
func RatioTotal(a int64, b int64) float64 {
	return float64(a) / (float64(a) + float64(b))
}

func logMetaCounter(mc *metaCounter, cs map[string]*counter) {
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

	log.Printf(
		theCtx.fmtStringF64,
		mc.name,
		vTotal,
		vDelta,
	)
}
