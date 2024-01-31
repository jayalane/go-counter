// -*- tab-width: 2 -*-

package counters

// this count_distro.go file is mostly a client of counters.go API - it bundles up a distribution into log 10 based buckets (really
// like 1, 2, 5, 10, 20, 50, 100, 200, 500, 1K, 2K, 5K, ... etc.

import (
	"fmt"
	"math"
)

type Resolution func (float64, int, string) string
var theResolution = HighRes

var units = []string{"f", "p", "n", "mi", "m", "", "k", "M", "G", "T", "P"}

var unitSort = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}

func LowRes(shortVal float64, size3 int, unit string) string {
	s := ""
	if shortVal >= 500 {
		s = "500" + unit + "-1" + units[size3+6]
	} else if shortVal >= 200 {
		s = "200" + unit + "-500" + unit
	} else if shortVal >= 100 {
		s = "100" + unit + "-200" + unit
	} else if shortVal >= 50 {
		s = "050" + unit + "-100" + unit
	} else if shortVal >= 20 {
		s = "020" + unit + "-50" + unit
	} else if shortVal >= 10 {
		s = "010" + unit + "-20" + unit
	} else if shortVal >= 5 {
		s = "005" + unit + "-10" + unit
	} else if shortVal >= 2 {
		s = "002" + unit + "-5" + unit
	} else if shortVal >= 1 {
		s = "001" + unit + "-2" + unit
	}

	return s
}

func MediumRes(shortVal float64, size3 int, unit string) string {
	s := ""

	if shortVal >= 900 {
		s = "900" + unit + "-1" + units[size3+6]
	} else if shortVal >= 800 {
		s = "800" + unit + "-900" + unit
	} else if shortVal >= 700 {
		s = "700" + unit + "-800" + unit
	} else if shortVal >= 600 {
		s = "600" + unit + "-700" + unit
	}	else if shortVal >= 500 {
		s = "500" + unit + "-600" + unit
	} else if shortVal >= 400 {
		s = "400" + unit + "-500" + unit
	} else if shortVal >= 300 {
		s = "300" + unit + "-400" + unit
	} else if shortVal >= 200 {
		s = "200" + unit + "-300" + unit
	} else if shortVal >= 100 {
		s = "100" + unit + "-200" + unit
	} else if shortVal >= 90 {
		s = "090" + unit + "-100" + unit
	} else if shortVal >= 80 {
		s = "080" + unit + "-90" + unit
	} else if shortVal >= 70 {
		s = "070" + unit + "-80" + unit
	} else if shortVal >= 60 {
		s = "060" + unit + "-70" + unit
	} else if shortVal >= 50 {
		s = "050" + unit + "-60" + unit
	} else if shortVal >= 40 {
		s = "040" + unit + "-50" + unit
	} else if shortVal >= 30 {
		s = "030" + unit + "-40" + unit
	} else if shortVal >= 20 {
		s = "020" + unit + "-30" + unit
	} else if shortVal >= 10 {
		s = "010" + unit + "-20" + unit
	} else if shortVal >= 9 {
		s = "009" + unit + "-10" + unit
	} else if shortVal >= 8 {
		s = "008" + unit + "-9" + unit
	} else if shortVal >= 7 {
		s = "007" + unit + "-8" + unit
	} else if shortVal >= 6 {
		s = "006" + unit + "-7" + unit
	} else if shortVal >= 5 {
		s = "005" + unit + "-6" + unit
	} else if shortVal >= 4 {
		s = "004" + unit + "-5" + unit
	} else if shortVal >= 3 {
		s = "003" + unit + "-4" + unit
	} else if shortVal >= 2 {
		s = "002" + unit + "-3" + unit
	} else if shortVal >= 1 {
		s = "001" + unit + "-2" + unit
	}
	return s
}


func HighRes(shortVal float64, size3 int, unit string) string {
	s := ""

	if shortVal >= 990 {
		s = "990" + unit + "-1" + units[size3+6]
		return s
	}
	if shortVal >= 99.99 {
		for x := float64(980); x >= 100; x = x - 10 {
			if shortVal >= x {
				s = fmt.Sprintf("%03.0f", x) + unit + "-" + fmt.Sprintf("%.0f", x + 10) + unit
				return s
			}
		}
	}
	if shortVal >= 9.99 {
		for x := float64(99); x >= 10; x = x - 1.0 {
			if shortVal >= x {
				s = fmt.Sprintf("%03.0f", x) + unit + "-" + fmt.Sprintf("%.0f", x + 1) + unit
				return s
			}
		}
	}
	for x := float64(9.9); x >= 1; x = x - 0.1 {
		if shortVal >= x {
			s = fmt.Sprintf("%05.1f", x) + unit + "-" + fmt.Sprintf("%.1f", x + 0.1) + unit
			return s
		}
	}
	return "err"
}

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
	size3 := int(math.Floor(float64(size) / 3.0)) // int division rounding left
	shortVal := value / math.Pow(10, float64(size3*3))
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

// MarkDistributionSyncSuffix is the fastest API
// One line does it all.
func MarkDistributionSyncSuffix(name string, value float64, suffix string) {
	derived := deriveDistName(name, value)
	IncrSyncSuffix(derived, suffix)
}
