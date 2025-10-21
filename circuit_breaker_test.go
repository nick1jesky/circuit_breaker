package circuitbreaker

// topics:
// 1) base functionality
// 2) thread safety
// 3) custom thresholds

import (
	"sync"
	"testing"
	"time"
)

// base functionality

func TestNewCircuitBreakerWithConfig(t *testing.T) {
	cfg := &CircuitBreakerCfg{
		FailureThreshold: 3,
		SuccessThreshold: 5,
		OpenTimeout:      10 * time.Second,
	}

	failureThreshold := NewInt64Threshold(int64(cfg.FailureThreshold))
	successThreshold := NewInt64Threshold(int64(cfg.SuccessThreshold))

	cb := NewCircuitBreaker(failureThreshold, successThreshold, cfg.OpenTimeout)
	if cb == nil {
		t.Fatal("Expected circuit breaker instance, got nil")
	}

	if state := cb.State(); state != StateClosed {
		t.Fatalf("Expected %s, got %s", StateClosed, state)
	}
}

func TestAllow(t *testing.T) {
	cb := NewCircuitBreaker(
		NewInt64Threshold(2),
		NewInt64Threshold(8),
		100*time.Millisecond,
	)

	if !cb.Allow() {
		t.Error("Expected Allow() to return true in closed state")
	}

	cb.RecordFailure()
	cb.RecordFailure()

	if cb.Allow() {
		t.Error("Expected Allow() to return false in open state")
	}

	time.Sleep(150 * time.Millisecond)
	if !cb.Allow() {
		t.Error("Expected Allow() to return true in half-open state")
	}
}

func TestRecordStates(t *testing.T) {
	cb := NewCircuitBreaker(
		NewInt64Threshold(1),
		NewInt64Threshold(1),
		100*time.Millisecond,
	)

	cb.RecordSuccess()
	if state := cb.State(); state != StateClosed {
		t.Errorf("Expected state %s, got %s", StateClosed, state)
	}

	cb.RecordFailure()
	time.Sleep(150 * time.Millisecond)
	cb.State()

	cb.RecordSuccess()
	if state := cb.State(); state != StateClosed {
		t.Errorf("Expected state %s, got %s", StateClosed, state)
	}

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()
	if state := cb.State(); state != StateOpened {
		t.Errorf("Expected state %s, got %s", StateOpened, state)
	}
}

// thread safety

func TestConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(
		NewInt64Threshold(100),
		NewInt64Threshold(100),
		100*time.Millisecond,
	)

	var wg sync.WaitGroup
	iterations := 10000
	goroutines := 10

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range iterations {
				if cb.Allow() {
					if j%20 == 0 {
						cb.RecordFailure()
					} else {
						cb.RecordSuccess()
					}
				}

				if j%1500 == 0 {
					cb.UpdateValues(
						NewInt64Threshold(20),
						NewInt64Threshold(200),
						500*time.Millisecond,
					)
				}

				if j%1300 == 0 {
					cb.UpdateValues(
						NewInt64Threshold(100),
						NewInt64Threshold(100),
						100*time.Millisecond,
					)
				}

				if j%50 == 0 {
					_ = cb.State()
				}
			}
		}()
	}

	wg.Wait()

	state := cb.State()
	if state != StateClosed && state != StateOpened && state != StateHalfOpen {
		t.Errorf("Invalid final state: %s", state)
	}
}

// custom thresholds - SlidingWindowThreshold

func TestSlidingWindowThreshold(t *testing.T) {
	t.Run("Basic SlidingWindowThreshold", func(t *testing.T) {
		windowSize := 100 * time.Millisecond
		maxFailures := 3
		threshold := NewSlidingWindowThreshold(windowSize, maxFailures, "test-window")

		if threshold.Check(nil) {
			t.Error("Expected Check to return false initially")
		}

		threshold.RecordFailure()
		threshold.RecordFailure()
		if threshold.Check(nil) {
			t.Error("Expected Check to return false with 2 failures")
		}

		threshold.RecordFailure()
		if !threshold.Check(nil) {
			t.Error("Expected Check to return true with 3 failures")
		}
	})

	t.Run("Sliding Window Expiration", func(t *testing.T) {
		windowSize := 50 * time.Millisecond
		maxFailures := 2
		threshold := NewSlidingWindowThreshold(windowSize, maxFailures, "expiration-test")

		threshold.RecordFailure()
		threshold.RecordFailure()

		if !threshold.Check(nil) {
			t.Error("Expected Check to return true with 2 failures")
		}

		time.Sleep(60 * time.Millisecond)

		if threshold.Check(nil) {
			t.Error("Expected Check to return false after window expiration")
		}
	})

	t.Run("Mixed Old and New Failures", func(t *testing.T) {
		windowSize := 100 * time.Millisecond
		maxFailures := 3
		threshold := NewSlidingWindowThreshold(windowSize, maxFailures, "mixed-test")

		threshold.RecordFailure()
		threshold.RecordFailure()

		time.Sleep(60 * time.Millisecond)

		threshold.RecordFailure()

		if !threshold.Check(nil) {
			t.Error("Expected Check to return true with mixed old and new failures")
		}
	})

	t.Run("GetCurrentFailures", func(t *testing.T) {
		windowSize := 100 * time.Millisecond
		maxFailures := 2
		threshold := NewSlidingWindowThreshold(windowSize, maxFailures, "current-test")

		if count := threshold.GetCurrentFailures(); count != 0 {
			t.Errorf("Expected 0 failures initially, got %d", count)
		}

		threshold.RecordFailure()
		if count := threshold.GetCurrentFailures(); count != 1 {
			t.Errorf("Expected 1 failure, got %d", count)
		}

		threshold.RecordFailure()
		if count := threshold.GetCurrentFailures(); count != 2 {
			t.Errorf("Expected 2 failures, got %d", count)
		}
	})

	t.Run("String and GetThreshold", func(t *testing.T) {
		windowSize := 100 * time.Millisecond
		maxFailures := 2
		name := "string-test"
		threshold := NewSlidingWindowThreshold(windowSize, maxFailures, name)

		expectedString := "SlidingWindowThreshold: " + name
		if actual := threshold.String(); actual != expectedString {
			t.Errorf("Expected String() to return '%s', got '%s'", expectedString, actual)
		}

		if th := threshold.GetThreshold(); th != threshold {
			t.Error("Expected GetThreshold() to return the same instance")
		}
	})

	t.Run("SlidingWindowThreshold Thread Safety", func(t *testing.T) {
		windowSize := 200 * time.Millisecond
		maxFailures := 10
		threshold := NewSlidingWindowThreshold(windowSize, maxFailures, "concurrent-test")

		var wg sync.WaitGroup
		iterations := 100

		for range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					threshold.RecordFailure()
					_ = threshold.Check(nil)
					_ = threshold.GetCurrentFailures()
				}
			}()
		}

		wg.Wait()

		finalFailures := threshold.GetCurrentFailures()
		if finalFailures < 0 || finalFailures > 500 {
			t.Errorf("Unexpected final failure count: %d", finalFailures)
		}
	})
}

func TestCircuitBreakerWithSlidingWindowThreshold(t *testing.T) {
	windowSize := 100 * time.Millisecond
	maxFailures := 2
	slidingThreshold := NewSlidingWindowThreshold(windowSize, maxFailures, "cb-integration")

	cb := NewCircuitBreaker(
		slidingThreshold,
		NewInt64Threshold(2),
		50*time.Millisecond,
	)

	if state := cb.State(); state != StateClosed {
		t.Errorf("Expected state %s, got %s", StateClosed, state)
	}

	slidingThreshold.RecordFailure()
	cb.RecordFailure()

	if state := cb.State(); state != StateClosed {
		t.Errorf("Expected state %s after 1 failure, got %s", StateClosed, state)
	}

	slidingThreshold.RecordFailure()
	cb.RecordFailure()

	if state := cb.State(); state != StateOpened {
		t.Errorf("Expected state %s after 2 failures, got %s", StateOpened, state)
	}
}

func TestSlidingWindowThresholdWithTimeControl(t *testing.T) {

	windowSize := 50 * time.Millisecond
	maxFailures := 2
	threshold := NewSlidingWindowThreshold(windowSize, maxFailures, "time-control-test")

	threshold.RecordFailure()

	time.Sleep(20 * time.Millisecond)
	threshold.RecordFailure()

	if !threshold.Check(nil) {
		t.Error("Expected Check to return true with 2 recent failures")
	}

	time.Sleep(40 * time.Millisecond)

	if threshold.Check(nil) {
		t.Error("Expected Check to return false after first failure expired")
	}

	if count := threshold.GetCurrentFailures(); count != 1 {
		t.Errorf("Expected 1 current failure, got %d", count)
	}
}
