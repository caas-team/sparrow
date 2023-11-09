package db

import (
	"github.com/caas-team/sparrow/pkg/checks"
	"reflect"
	"sync"
	"testing"
)

func TestInMemory_Save(t *testing.T) {
	type fields struct {
		data map[string]*checks.Result
		mu   sync.Mutex
	}
	type args struct {
		result checks.ResultDTO
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{name: "Saves without error", fields: fields{data: make(map[string]*checks.Result), mu: sync.Mutex{}}, args: args{result: checks.ResultDTO{Name: "Test", Result: &checks.Result{Data: 0}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &InMemory{
				data: tt.fields.data,
				mu:   tt.fields.mu,
			}
			i.Save(tt.args.result)
			if val, ok := i.data[tt.args.result.Name]; !ok {
				t.Errorf("Expected to find key %s in map", tt.args.result.Name)
			} else {
				if !reflect.DeepEqual(val, tt.args.result.Result) {
					t.Errorf("Expected val to be %v but got: %v", val, tt.args.result.Result)
				}
			}
		})
	}
}

func TestNewInMemory(t *testing.T) {
	tests := []struct {
		name string
		want *InMemory
	}{
		{name: "Creates without nil pointers", want: &InMemory{data: make(map[string]*checks.Result), mu: sync.Mutex{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInMemory(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInMemory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemory_Get(t *testing.T) {
	type fields struct {
		data map[string]*checks.Result
		mu   sync.Mutex
	}
	type args struct {
		check string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   checks.Result
	}{
		{name: "Can get value", fields: fields{mu: sync.Mutex{}, data: map[string]*checks.Result{
			"alpha": {Data: 0},
			"beta":  {Data: 1},
		}}, want: checks.Result{Data: 1}, args: args{check: "beta"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &InMemory{
				data: tt.fields.data,
				mu:   tt.fields.mu,
			}
			if got := i.Get(tt.args.check); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemory_List(t *testing.T) {
	type fields struct {
		data map[string]*checks.Result
		mu   sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]*checks.Result
	}{
		{name: "Lists all entries", fields: fields{
			data: map[string]*checks.Result{
				"alpha": {Data: 0},
				"beta":  {Data: 1},
			},
			mu: sync.Mutex{},
		}, want: map[string]*checks.Result{
			"alpha": {Data: 0},
			"beta":  {Data: 1},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &InMemory{
				data: tt.fields.data,
				mu:   tt.fields.mu,
			}
			if got := i.List(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("List() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemory_ListThreadsafe(t *testing.T) {
	db := NewInMemory()
	db.Save(checks.ResultDTO{Name: "alpha", Result: &checks.Result{Data: 0}})
	db.Save(checks.ResultDTO{Name: "beta", Result: &checks.Result{Data: 1}})

	got := db.List()
	if len(got) != 2 {
		t.Errorf("Expected 2 entries but got %d", len(got))
	}

	got["alpha"] = &checks.Result{Data: 50}

	newGot := db.List()
	if newGot["alpha"].Data != 0 {
		t.Errorf("Expected alpha to be 0 but got %d", newGot["alpha"].Data)
	}

}
