package counters

import (
	"sync"
	"testing"
)

func TestGetOrMakeAndIncrCounter_RepeatedConcurrent(t *testing.T) {
	for range 1000 {
		TestGetOrMakeAndIncrCounter_Concurrent(t)
		// Reset the counters after each run to ensure a clean state for the next iteration.
		theCtx.ctxLock.Lock()
		// Resetting countersByName, where the counter from the concurrent test should reside.
		theCtx.countersByName = make(map[string]*counter)
		theCtx.ctxLock.Unlock()
	}
}

func TestGetOrMakeAndIncrCounter_Concurrent(t *testing.T) {
	InitCounters() // Initialize the counters infrastructure

	const (
		metricName           = "test_metric"
		numRoutines          = 10
		incrementsPerRoutine = 1000
	)

	var start sync.WaitGroup

	start.Add(1)

	var wg sync.WaitGroup

	wg.Add(numRoutines)

	for range numRoutines {
		go func() {
			defer wg.Done()
			start.Wait() // Wait for all goroutines to start

			for range incrementsPerRoutine {
				getOrMakeAndIncrCounter(metricName, "", 1)
			}
		}()
	}

	start.Done() // Start all goroutines
	wg.Wait()    // Wait for all goroutines to finish

	theCtx.ctxLock.RLock()
	defer theCtx.ctxLock.RUnlock()

	c, ok := theCtx.countersByName[metricName]
	if !ok {
		t.Fatalf("Metric %s not found", metricName)
	}

	expectedCount := int64(numRoutines * incrementsPerRoutine)

	if c.data != expectedCount {
		t.Errorf("Expected count %d, got %d", expectedCount, c.data)
	}
}
