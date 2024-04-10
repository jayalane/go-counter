// -*- tab-width: 2 -*-

package counters

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type t struct {
	name             string
	value            float64
	lowResDerived    string
	mediumResDerived string
	highResDerived   string
}

var testsDerived = []t{
	t{
		name:             "test",
		value:            1113.0,
		lowResDerived:    "testg[001k-2k]",
		mediumResDerived: "testg[001k-2k]",
		highResDerived:   "testg[001.1k-1.2k]",
	},
	t{
		name:             "test",
		value:            0.0,
		lowResDerived:    "test [zero]",
		mediumResDerived: "test [zero]",
		highResDerived:   "test [zero]",
	},
	t{
		name:             "test",
		value:            2113.0,
		lowResDerived:    "testg[002k-5k]",
		mediumResDerived: "testg[002k-3k]",
		highResDerived:   "testg[002.1k-2.2k]",
	},
	t{
		name:             "test",
		value:            5113.0,
		lowResDerived:    "testg[005k-10k]",
		mediumResDerived: "testg[005k-6k]",
		highResDerived:   "testg[005.1k-5.2k]",
	},
	t{
		name:             "test",
		value:            15113.0,
		lowResDerived:    "testg[010k-20k]",
		mediumResDerived: "testg[010k-20k]",
		highResDerived:   "testg[015k-16k]",
	},
	t{
		name:             "test",
		value:            45113.0,
		lowResDerived:    "testg[020k-50k]",
		mediumResDerived: "testg[040k-50k]",
		highResDerived:   "testg[045k-46k]",
	},
	t{
		name:             "test",
		value:            95113.0,
		lowResDerived:    "testg[050k-100k]",
		mediumResDerived: "testg[090k-100k]",
		highResDerived:   "testg[095k-96k]",
	},
	t{
		name:             "test2",
		value:            113.0,
		lowResDerived:    "test2f[100-200]",
		mediumResDerived: "test2f[100-200]",
		highResDerived:   "test2f[110-120]",
	},
	t{
		name:             "test2",
		value:            213.0,
		lowResDerived:    "test2f[200-500]",
		mediumResDerived: "test2f[200-300]",
		highResDerived:   "test2f[210-220]",
	},
	t{
		name:             "test2",
		value:            0.23,
		lowResDerived:    "test2e[200m-500m]",
		mediumResDerived: "test2e[200m-300m]",
		highResDerived:   "test2e[230m-240m]",
	},
	t{
		name:             "test2",
		value:            0.831,
		lowResDerived:    "test2e[500m-1]",
		mediumResDerived: "test2e[800m-900m]",
		highResDerived:   "test2e[830m-840m]",
	},
	t{
		name:             "test2",
		value:            0.00083,
		lowResDerived:    "test2d[500mi-1m]",
		mediumResDerived: "test2d[800mi-900mi]",
		highResDerived:   "test2d[830mi-840mi]",
	},
}

func TestDerivDistNameLow(t *testing.T) {
	for _, te := range testsDerived {
		SetResolution(LowRes)
		s := deriveDistName(te.name, te.value)
		if s != te.lowResDerived {
			fmt.Println("Got", s, "Expected", te.lowResDerived, "from", te.value)
			t.Fail()
		}
	}
}

func TestDerivDistMedium(t *testing.T) {
	for _, te := range testsDerived {
		SetResolution(MediumRes)
		s := deriveDistName(te.name, te.value)
		if s != te.mediumResDerived {
			fmt.Println("Got", s, "Expected", te.mediumResDerived, "from", te.value)
			t.Fail()
		}
	}
}

func TestDerivDistNameHigh(t *testing.T) {
	for _, te := range testsDerived {
		SetResolution(HighRes)
		s := deriveDistName(te.name, te.value)
		if s != te.highResDerived {
			fmt.Println("Got", s, "Expected", te.highResDerived, "from", te.value)
			t.Fail()
		}
	}
}

func TestDerivMarkDist(t *testing.T) {
	InitCounters()
	SetLogInterval(1)
	for _, te := range testsDerived {
		MarkDistribution(te.name, te.value)
	}
	TimeFuncRun("chris", func() {
		time.Sleep(256 * time.Millisecond)
	})
	LogCounters()
}

func TestErr(t *testing.T) {
	InitCounters()
	SetLogInterval(1)
	for i := 0; i < 1_000_000; i++ {
		MarkDistribution("seeking_err", rand.Float64())
		MarkDistribution("seeking_err", 1.0/rand.Float64())
		MarkDistribution("seeking_err", 0)
	}
	LogCounters()
}
