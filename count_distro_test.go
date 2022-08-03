// -*- tab-width: 2 -*-

package counters

import (
	"fmt"
	"testing"
)

type t struct {
	name    string
	value   float64
	derived string
}

var testsDerived = []t{
	t{
		name:    "test",
		value:   1113.0,
		derived: "test[1k-2k]",
	},
	t{
		name:    "test",
		value:   2113.0,
		derived: "test[2k-5k]",
	},
	t{
		name:    "test",
		value:   5113.0,
		derived: "test[5k-10k]",
	},
	t{
		name:    "test",
		value:   15113.0,
		derived: "test[10k-20k]",
	},
	t{
		name:    "test",
		value:   45113.0,
		derived: "test[20k-50k]",
	},
	t{
		name:    "test",
		value:   95113.0,
		derived: "test[50k-100k]",
	},
	t{
		name:    "test2",
		value:   113.0,
		derived: "test2[100-200]",
	},
	t{
		name:    "test2",
		value:   213.0,
		derived: "test2[200-500]",
	},
	t{
		name:    "test2",
		value:   0.23,
		derived: "test2[200m-500m]",
	},
	t{
		name:    "test2",
		value:   0.83,
		derived: "test2[500m-1]",
	},
	t{
		name:    "test2",
		value:   0.00083,
		derived: "test2[500mi-1m]",
	},
}

func TestDerivDistName(t *testing.T) {
	for _, te := range testsDerived {
		s := deriveDistName(te.name, te.value)
		if s != te.derived {
			fmt.Println("Got", s, "Expected", te.derived, "from", te.value)
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
	LogCounters()
}
