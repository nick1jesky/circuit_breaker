package circuitbreaker

// - is an interface for all types of threshold values
type CustomThreshold interface {
	Check(value any) bool
	GetThreshold() any
}

// - defines interface for controll circuit breaker state switching
type Switch interface {
	Check(value any) bool
}

// - is a switch for custom thresholds
type CustomSwitch struct {
	threshold CustomThreshold
}

// - check
func (s CustomSwitch) Check(value any) bool {
	return s.threshold.Check(value)
}

// - choses realisation of Switch
func ChooseSwitch(threshold CustomThreshold) Switch {
	return CustomSwitch{threshold: threshold}
}

// Custom examples of thresholds

type Int64Threshold struct {
	threshold int64
}

func NewInt64Threshold(threshold int64) *Int64Threshold {
	return &Int64Threshold{threshold: threshold}
}

func (t *Int64Threshold) Check(value any) bool {
	switch v := value.(type) {
	case int64:
		return v >= t.threshold
	case int:
		return int64(v) >= t.threshold
	default:
		return false
	}
}

func (t *Int64Threshold) GetThreshold() any {
	return t.threshold
}

type Float64Threshold struct {
	threshold float64
}

func NewFloat64Threshold(threshold float64) *Float64Threshold {
	return &Float64Threshold{threshold: threshold}
}

func (t *Float64Threshold) Check(value any) bool {
	switch v := value.(type) {
	case float64:
		return v >= t.threshold
	case float32:
		return float64(v) >= t.threshold
	case int:
		return float64(v) >= t.threshold
	case int64:
		return float64(v) >= t.threshold
	default:
		return false
	}
}

func (t *Float64Threshold) GetThreshold() any {
	return t.threshold
}
