// -*- tab-width: 2 -*-

package counters

// this count_distro.go file is mostly a client of counters.go API - it bundles up a distribution into log 10 based buckets (really
// like 1, 2, 5, 10, 20, 50, 100, 200, 500, 1K, 2K, 5K, ... etc.

import (
	"math"
)

var units = []string{"f", "p", "n", "mi", "m", "", "k", "M", "G", "T", "P"}

func deriveDistName(name string, value float64) string {
	res := ""
	sign := ""
	if value < 0 {
		sign = "-"
		value = 0 - value
	}
	size := int(math.Floor(math.Log10(value)))
	size3 := int(math.Floor(float64(size) / 3.0)) // int division rounding left
	shortVal := value / math.Pow(10, float64(size3*3))
	unit := ""
	if size3+5 < len(units)-1 && 0 <= size3+5 {
		unit = units[size3+5]
	} else {
		unit = "handleOddSizes(string, value)"
	}
	s := ""
	if shortVal >= 500 {
		s = "500" + unit + "-1" + units[size3+6]
	} else if shortVal >= 200 {
		s = "200" + unit + "-500" + unit
	} else if shortVal >= 100 {
		s = "100" + unit + "-200" + unit
	} else if shortVal >= 50 {
		s = "50" + unit + "-100" + unit
	} else if shortVal >= 20 {
		s = "20" + unit + "-50" + unit
	} else if shortVal >= 10 {
		s = "10" + unit + "-20" + unit
	} else if shortVal >= 5 {
		s = "5" + unit + "-10" + unit
	} else if shortVal >= 2 {
		s = "2" + unit + "-5" + unit
	} else if shortVal >= 1 {
		s = "1" + unit + "-2" + unit
	}
	res = name + sign + "[" + s + "]"
	return res
}

// MarkDistribution transforms the name and value
// to a histogram bucket and marks it
func MarkDistribution(name string, value float64) {
	derived := deriveDistName(name, value)
	Incr(derived)
}

// MarkDistributionSuffix transforms the name and value to a histogram
// bucket and marks it, taking a suffix for efficiency
func MarkDistributionSuffix(name string, value float64, suffix string) {
	derived := deriveDistName(name, value)
	IncrSuffix(derived, suffix)
}

// MarkDistributionSync is the faster API
// One line does it all.
func MarkDistributionSync(name string, value float64) {
	derived := deriveDistName(name, value)
	IncrSync(derived)
}