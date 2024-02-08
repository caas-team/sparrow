package targets

import "errors"

// ErrInvalidCheckInterval is returned when the check interval is invalid
var ErrInvalidCheckInterval = errors.New("invalid check interval")

// ErrInvalidRegistrationInterval is returned when the registration interval is invalid
var ErrInvalidRegistrationInterval = errors.New("invalid registration interval")

// ErrInvalidUnhealthyThreshold is returned when the unhealthy threshold is invalid
var ErrInvalidUnhealthyThreshold = errors.New("invalid unhealthy threshold")
