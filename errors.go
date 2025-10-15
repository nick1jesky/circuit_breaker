package circuitbreaker

import "errors"

var (
	ErrUnsupporterType = errors.New("unsupported type")
	ErrNotImplemented  = errors.New("not implemented")
)
