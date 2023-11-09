package db

import (
	"sync"

	"github.com/caas-team/sparrow/pkg/checks"
)

type DB interface {
	Save(result checks.ResultDTO)
	Get(check string) checks.Result
	List() map[string]*checks.Result
}

type InMemory struct {
	// if we want to save a timeseries
	// we can use a map of ringbuffers instead of a single value
	// this ensures that we can save the last N results, where N is the size of the ringbuffer
	// without having to worry about the size of the map
	data map[string]*checks.Result
	mu   sync.Mutex
}

// NewInMemory creates a new in-memory database
func NewInMemory() *InMemory {
	return &InMemory{
		data: make(map[string]*checks.Result),
		mu:   sync.Mutex{},
	}
}

func (i *InMemory) Save(result checks.ResultDTO) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.data[result.Name] = result.Result
}

func (i *InMemory) Get(check string) checks.Result {
	i.mu.Lock()
	defer i.mu.Unlock()
	return *i.data[check]
}

// Returns a copy of the map
func (i *InMemory) List() map[string]*checks.Result {
	results := make(map[string]*checks.Result)
	i.mu.Lock()
	for k, v := range i.data {
		results[k] = v
	}
	defer i.mu.Unlock()
	return results
}
