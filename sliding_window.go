package circuitbreaker

import (
	"sync"
	"time"
)

type SlidingWindowThreshold struct {
	windowSize  time.Duration
	maxFailures int
	name        string

	mu           sync.RWMutex
	failureTimes []time.Time
}

func NewSlidingWindowThreshold(
  windowSize time.Duration, 
  maxFailures int, 
  name string,
) *SlidingWindowThreshold {
	return &SlidingWindowThreshold{
		windowSize:   windowSize,
		maxFailures:  maxFailures,
		name:         name,
		failureTimes: make([]time.Time, 0),
	}
}

func (sw *SlidingWindowThreshold) Check(value any) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.windowSize)

	validFailures := make([]time.Time, 0)
	for _, ft := range sw.failureTimes {
		if ft.After(windowStart) {
			validFailures = append(validFailures, ft)
		}
	}
	sw.failureTimes = validFailures

	return len(sw.failureTimes) >= sw.maxFailures
}

func (sw *SlidingWindowThreshold) GetThreshold() any {
	return sw
}

func (sw *SlidingWindowThreshold) String() string {
	return "SlidingWindowThreshold: " + sw.name
}

func (sw *SlidingWindowThreshold) RecordFailure() {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.failureTimes = append(sw.failureTimes, time.Now())
}

func (sw *SlidingWindowThreshold) GetCurrentFailures() int {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	return len(sw.failureTimes)
}
