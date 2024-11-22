// -*- tab-width: 2 -*-

package counters

// this gc.go extracts the run time counters and puts them into this package.

import (
	"fmt"
	"math"
	"runtime/metrics"
	"strings"
)

func checkRuntime() {
	ms := metrics.All()
	// next 10 lines from https://pkg.go.dev/runtime/metrics#example-Read-ReadingAllMetrics
	// Create a sample for each metric.
	samples := make([]metrics.Sample, len(ms))
	desc := make(map[string]metrics.Description)

	for i := range samples {
		samples[i].Name = ms[i].Name
		desc[ms[i].Name] = ms[i]
	}

	// Sample the metrics.
	metrics.Read(samples)

	for _, m := range samples {
		name, value := m.Name, m.Value
		name = strings.ReplaceAll(name, "/", "_")
		name = strings.ReplaceAll(name, ":", "_")

		if name[0] == '_' {
			name = name[1:]
		}

		name = "0_" + name

		if value.Kind() == metrics.KindUint64 && desc[m.Name].Cumulative { //nolint:gocritic
			if value.Uint64() > math.MaxInt64 {
				fmt.Println("Skipping large metric", value.Uint64(), "9223372036854775807") // string for 32 bit architectures

				continue
			}

			IncrDeltaSuffix(name, int64(value.Uint64()), "go-runtime") //nolint:gosec
		} else if value.Kind() == metrics.KindFloat64 {
			SetSuffix(name, value.Float64(), "go-runtime")
		} else if value.Kind() == metrics.KindUint64 {
			vv := float64(value.Uint64())
			SetSuffix(name, vv, "go-runtime")
		}
	}
}
