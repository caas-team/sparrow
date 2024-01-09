package config

import "time"

// ResultDTO is a data transfer object used to associate a check's name with its result.
type ResultDTO struct {
	Name   string
	Result *Result
}

// Result encapsulates the outcome of a check run.
type Result struct {
	// data contains performance metrics about the check run
	Data any `json:"data"`
	// Timestamp is the UTC time the check was run
	Timestamp time.Time `json:"timestamp"`
	// Err should be nil if the check ran successfully indicating the check is "healthy"
	// if the check failed, this should be an error message that will be logged and returned to an API user
	Err string `json:"error"`
}
