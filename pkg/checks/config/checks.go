package config

import (
	"net/http"
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/helper"
)

var (
	// BasicRetryConfig provides a default configuration for the retry mechanism
	DefaultRetry = helper.RetryConfig{
		Count: 3,
		Delay: time.Second,
	}
)

// CheckBase is a struct providing common fields used by implementations of the Check interface.
// It serves as a foundational structure that should be embedded in specific check implementations.
type CheckBase struct {
	Mu      sync.Mutex
	CResult chan<- Result
	Done    chan bool
	Client  *http.Client
}

// GlobalTarget includes the basic information regarding
// other Sparrow instances, which this Sparrow can communicate with.
type GlobalTarget struct {
	Url      string    `json:"url"`
	LastSeen time.Time `json:"lastSeen"`
}
