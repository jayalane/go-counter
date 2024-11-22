// nolint:mnd -*- tab-width: 2 -*-

package counters

// this count_distro.go file is mostly a client of counters.go API - it bundles up a distribution into log 10 based buckets (really
// like 1, 2, 5, 10, 20, 50, 100, 200, 500, 1K, 2K, 5K, ... etc.

import (
	"fmt"
	"math"
)

// Resolution is a function type used with some
// predefined constants to allow the library
// user to choose histogram bucket resolution.
type Resolution func(float64, int, string) string

var theResolution = HighRes

var units = []string{"f", "p", "n", "mi", "m", "", "k", "M", "G", "T", "P"}

var unitSort = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}

// LowRes is a bucketing constant for 1/2/5/10/20/50 style buckets.
func LowRes(shortVal float64, size3 int, unit string) string {
	s := ""
	if shortVal >= 500 { // nolint:gocritic,nestif,mnd
		s = "500" + unit + "-1" + units[size3+6]
	} else if shortVal >= 200 { //nolint:mnd
		s = "200" + unit + "-500" + unit
	} else if shortVal >= 100 { //nolint:mnd
		s = "100" + unit + "-200" + unit
	} else if shortVal >= 50 { //nolint:mnd
		s = "050" + unit + "-100" + unit
	} else if shortVal >= 20 { //nolint:mnd
		s = "020" + unit + "-50" + unit
	} else if shortVal >= 10 { //nolint:mnd
		s = "010" + unit + "-20" + unit
	} else if shortVal >= 5 { //nolint:mnd
		s = "005" + unit + "-10" + unit
	} else if shortVal >= 2 { //nolint:mnd
		s = "002" + unit + "-5" + unit
	} else if shortVal >= 1 { //nolint:mnd
		s = "001" + unit + "-2" + unit
	}

	return s
}

// MediumRes is a bucketing constant for one digit of resolution.
func MediumRes(shortVal float64, size3 int, unit string) string { //nolint:cyclop
	s := ""

	if shortVal >= 900 { //nolint:gocritic,nestif,mnd
		s = "900" + unit + "-1" + units[size3+6]
	} else if shortVal >= 800 { //nolint:mnd
		s = "800" + unit + "-900" + unit
	} else if shortVal >= 700 { //nolint:mnd
		s = "700" + unit + "-800" + unit
	} else if shortVal >= 600 { //nolint:mnd
		s = "600" + unit + "-700" + unit
	} else if shortVal >= 500 { //nolint:mnd
		s = "500" + unit + "-600" + unit
	} else if shortVal >= 400 { //nolint:mnd
		s = "400" + unit + "-500" + unit
	} else if shortVal >= 300 { //nolint:mnd
		s = "300" + unit + "-400" + unit
	} else if shortVal >= 200 { //nolint:mnd
		s = "200" + unit + "-300" + unit
	} else if shortVal >= 100 { //nolint:mnd
		s = "100" + unit + "-200" + unit
	} else if shortVal >= 90 { //nolint:mnd
		s = "090" + unit + "-100" + unit
	} else if shortVal >= 80 { //nolint:mnd
		s = "080" + unit + "-90" + unit
	} else if shortVal >= 70 { //nolint:mnd
		s = "070" + unit + "-80" + unit
	} else if shortVal >= 60 { //nolint:mnd
		s = "060" + unit + "-70" + unit
	} else if shortVal >= 50 { //nolint:mnd
		s = "050" + unit + "-60" + unit
	} else if shortVal >= 40 { //nolint:mnd
		s = "040" + unit + "-50" + unit
	} else if shortVal >= 30 { //nolint:mnd
		s = "030" + unit + "-40" + unit
	} else if shortVal >= 20 { //nolint:mnd
		s = "020" + unit + "-30" + unit
	} else if shortVal >= 10 { //nolint:mnd
		s = "010" + unit + "-20" + unit
	} else if shortVal >= 9 { //nolint:mnd
		s = "009" + unit + "-10" + unit
	} else if shortVal >= 8 { //nolint:mnd
		s = "008" + unit + "-9" + unit
	} else if shortVal >= 7 { //nolint:mnd
		s = "007" + unit + "-8" + unit
	} else if shortVal >= 6 { //nolint:mnd
		s = "006" + unit + "-7" + unit
	} else if shortVal >= 5 { //nolint:mnd
		s = "005" + unit + "-6" + unit
	} else if shortVal >= 4 { //nolint:mnd
		s = "004" + unit + "-5" + unit
	} else if shortVal >= 3 { //nolint:mnd
		s = "003" + unit + "-4" + unit
	} else if shortVal >= 2 { //nolint:mnd
		s = "002" + unit + "-3" + unit
	} else if shortVal >= 1 { //nolint:mnd
		s = "001" + unit + "-2" + unit
	}

	return s
}

// HighRes is a constant for 2 sigfig of resolution.
func HighRes(shortVal float64, size3 int, unit string) string { //nolint:cyclop
	sign := ""

	if shortVal < 0 {
		sign = "-"
		shortVal = -1 * shortVal
	}

	s := ""

	if shortVal >= 990 { //nolint:mnd
		s = "990" + unit + "-1" + units[size3+6]

		return sign + s
	}

	if shortVal >= 99.5 { //nolint:mnd
		for x := float64(980); x >= 100; x -= 10 { //nolint:mnd
			if shortVal > x || math.Abs(shortVal-x) <= 5 {
				s = fmt.Sprintf("%03.0f", x) + unit + "-" + fmt.Sprintf("%.0f", x+10) + unit //nolint:mnd

				return sign + s
			}
		}
	}

	if shortVal >= 9.95 { //nolint:mnd
		for x := float64(99); x >= 10; x -= 1.0 { //nolint:mnd
			if shortVal > x || math.Abs(shortVal-x) <= 0.5 {
				s = fmt.Sprintf("%03.0f", x) + unit + "-" + fmt.Sprintf("%.0f", x+1) + unit

				return sign + s
			}
		}
	}

	for x := float64(99); x >= 10; x-- { //nolint:mnd
		if shortVal*10 > x || math.Abs(shortVal*10-x) <= 0.5 { //nolint:mnd
			s = fmt.Sprintf("%05.1f", x/10.0) + unit + "-" + fmt.Sprintf("%.1f", x/10.0+0.1) + unit //nolint:mnd

			return sign + s
		}
	}

	return "err"
}

// SetResolution lets the library caller to specify
// histogram bucket resolution.
func SetResolution(f Resolution) {
	theResolution = f
}

func deriveDistName(name string, value float64) string {
	if value == 0.0 {
		return name + " [zero]"
	}

	res := ""
	sign := ""

	if value < 0 {
		sign = "-"
		value = 0 - value
	}

	size := int(math.Floor(math.Log10(value)))
	size3 := int(math.Floor(float64(size) / 3.0))      //nolint:mnd
	shortVal := value / math.Pow(10, float64(size3*3)) //nolint:mnd
	unit := ""
	unitOrder := ""

	if size3+5 < len(units)-1 && 0 <= size3+5 {
		unit = units[size3+5]
		unitOrder = unitSort[size3+5]
	} else {
		unit = "handleOddSizes(string, value)"
	}

	s := theResolution(shortVal, size3, unit)
	res = name + sign + unitOrder + "[" + s + "]"

	return res
}

// MarkDistribution transforms the name and value
// to a histogram bucket and marks it.
func MarkDistribution(name string, value float64) {
	derived := deriveDistName(name, value)
	Incr(derived)
}

// MarkDistributionSuffix transforms the name and value to a histogram
// bucket and marks it, taking a suffix for efficiency.
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

// MarkDistributionSyncSuffix is the fastest API
// One line does it all.
func MarkDistributionSyncSuffix(name string, value float64, suffix string) {
	derived := deriveDistName(name, value)
	IncrSyncSuffix(derived, suffix)
}
