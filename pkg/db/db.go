package db

import (
	"sync"

	"github.com/caas-team/sparrow/pkg/checks"
)

type DB interface {
	Save(result checks.Result)
}

type InMemory struct {
	// if we want to save a timeseries
	// we can use a map of ringbuffers instead of a single value
	// this ensures that we can save the last N results, where N is the size of the ringbuffer
	// without having to worry about the size of the map
	data map[string]checks.Result
	mu   sync.Mutex
}

// NewInMemory creates a new in-memory database
func NewInMemory() *InMemory {
	return &InMemory{
		data: make(map[string]checks.Result),
		mu:   sync.Mutex{},
	}
}

func (i *InMemory) Save(result checks.Result) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.data[result.Check] = result
}
