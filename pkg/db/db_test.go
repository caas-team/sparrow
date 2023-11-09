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
