package circuitbreaker

import "time"

type CircuitBreakerCfg struct {
	FailureThreshold int
	SuccessThreshold int
	OpenTimeout      time.Duration
}
