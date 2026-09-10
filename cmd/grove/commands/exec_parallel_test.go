package commands

import (
	"sync"
	"testing"
	"time"
)

func TestRunParallel(t *testing.T) {
	t.Parallel()

	t.Run("returns all results with at most four targets in flight", func(t *testing.T) {
		t.Parallel()

		var mutex sync.Mutex
		inFlight, maximum := 0, 0
		started := make(chan struct{}, 8)
		release := make(chan struct{})
		done := make(chan []execResult, 1)
		go func() {
			done <- runParallel(make([]execTarget, 8), 4, func(execTarget) execResult {
				mutex.Lock()
				inFlight++
				maximum = max(maximum, inFlight)
				mutex.Unlock()

				started <- struct{}{}
				<-release

				mutex.Lock()
				inFlight--
				mutex.Unlock()

				return execResult{}
			})
		}()

		defer close(release)
		for range 4 {
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("four targets did not start concurrently")
			}
		}
		for range 8 {
			release <- struct{}{}
		}

		results := <-done
		if len(results) != 8 {
			t.Fatalf("expected eight results, got %d", len(results))
		}
		if maximum != 4 {
			t.Errorf("expected maximum concurrency four, got %d", maximum)
		}
	})
}
