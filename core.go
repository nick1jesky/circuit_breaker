package circuitbreaker

import (
	"sync"
	"time"
)

var (
	StateClosed   = "closed"
	StateOpened   = "open"
	StateHalfOpen = "half-open"
)

// - the circuit breaker with custom thresholds
type CircuitBreaker struct {
	mu sync.RWMutex

	failures  int64
	successes int64

	state           string
	lastStateChange time.Time

	failureThreshold CustomThreshold
	successThreshold CustomThreshold
	failureSwitch    Switch
	successSwitch    Switch

	openedTimeout time.Duration
}

// - is a constructor
func NewCircuitBreaker(
	failureThreshold,
	successThreshold CustomThreshold,
	openedTimeout time.Duration,
) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		failureSwitch:    ChooseSwitch(failureThreshold),
		successSwitch:    ChooseSwitch(successThreshold),
		openedTimeout:    openedTimeout,
		lastStateChange:  time.Now(),
	}
}

// - updates values of thresholds
func (cb *CircuitBreaker) UpdateValues(newFailure, newSuccess CustomThreshold, newTimeout time.Duration) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureThreshold = newFailure
	cb.successThreshold = newSuccess
	cb.failureSwitch = ChooseSwitch(newFailure)
	cb.successSwitch = ChooseSwitch(newSuccess)
	cb.openedTimeout = newTimeout
}

// resetCounters reset counters
func (cb *CircuitBreaker) resetCounters() {
	cb.failures = 0
	cb.successes = 0
}

// - checks is the operation allowed
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpened && time.Since(cb.lastStateChange) > cb.openedTimeout {
		cb.state = StateHalfOpen
		cb.lastStateChange = time.Now()
		cb.resetCounters()
	}

	return cb.state != StateOpened
}

// - calculates value to check threshold
func (cb *CircuitBreaker) calculateCheckValue(counter int64, threshold CustomThreshold) interface{} {
	switch threshold.(type) {
	case *Int64Threshold:
		return counter
	case *Float64Threshold:
		total := cb.successes + cb.failures
		if total == 0 {
			return 0.0
		}
		return float64(counter) / float64(total)
	default:
		return struct {
			Successes int64
			Failures  int64
			Total     int64
		}{
			Successes: cb.successes,
			Failures:  cb.failures,
			Total:     cb.successes + cb.failures,
		}
	}
}

// - records a success call
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.successes++
		cb.failures = 0

	case StateHalfOpen:
		cb.successes++
		cb.failures = 0

		checkValue := cb.calculateCheckValue(cb.successes, cb.successThreshold)
		if cb.successSwitch.Check(checkValue) {
			cb.state = StateClosed
			cb.lastStateChange = time.Now()
			cb.resetCounters()
		}

	case StateOpened:
		return
	}
}

// - records failure call
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failures++
		cb.successes = 0

		checkValue := cb.calculateCheckValue(cb.failures, cb.failureThreshold)
		if cb.failureSwitch.Check(checkValue) {
			cb.state = StateOpened
			cb.lastStateChange = time.Now()
			cb.resetCounters()
		}

	case StateHalfOpen:
		cb.state = StateOpened
		cb.lastStateChange = time.Now()
		cb.resetCounters()

	case StateOpened:
		return
	}
}

// - returns current state
func (cb *CircuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpened && time.Since(cb.lastStateChange) > cb.openedTimeout {
		cb.state = StateHalfOpen
		cb.lastStateChange = time.Now()
		cb.resetCounters()
	}
	return cb.state
}
