package db

import (
	"sync"

	"github.com/caas-team/sparrow/pkg/checks"
)

type DB interface {
	Save(result checks.ResultDTO)
	Get(check string) (result checks.Result, ok bool)
	List() map[string]checks.Result
}

var _ DB = (*InMemory)(nil)

type InMemory struct {
	// if we want to save a timeseries
	// we can use a map of ringbuffers instead of a single value
	// this ensures that we can save the last N results, where N is the size of the ringbuffer
	// without having to worry about the size of the map
	data sync.Map
}

// NewInMemory creates a new in-memory database
func NewInMemory() *InMemory {
	return &InMemory{
		data: sync.Map{},
	}
}

func (i *InMemory) Save(result checks.ResultDTO) {
	i.data.Store(result.Name, result.Result)
}

func (i *InMemory) Get(check string) (checks.Result, bool) {
	tmp, ok := i.data.Load(check)
	if !ok {
		return checks.Result{}, false
	}
	// this should not fail, otherwise this will panic
	result := tmp.(*checks.Result)

	return *result, true

}

// Returns a copy of the map
func (i *InMemory) List() map[string]checks.Result {
	results := make(map[string]checks.Result)
	i.data.Range(func(key, value any) bool {
		// this assertion should not fail, unless we have a bug somewhere
		check := key.(string)
		result := value.(*checks.Result)

		results[check] = *result
		return true
	})

	return results
}
