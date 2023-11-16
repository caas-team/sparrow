package api

import (
	"net/http"
	"sync"
	"testing"
)

func Test_routingTree_add(t *testing.T) {
	type fields struct {
		tree map[string]map[string]http.HandlerFunc
		mu   sync.RWMutex
	}
	type args struct {
		meth    string
		path    string
		handler http.HandlerFunc
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{name: "can add handler", fields: fields{tree: map[string]map[string]http.HandlerFunc{}, mu: sync.RWMutex{}}, args: args{meth: "GET", path: "/test", handler: func(w http.ResponseWriter, r *http.Request) {}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RoutingTree{
				tree: tt.fields.tree,
				mu:   tt.fields.mu,
			}
			r.Add(tt.args.meth, tt.args.path, tt.args.handler)

			if _, ok := r.tree[tt.args.meth][tt.args.path]; !ok {
				t.Errorf("routingTree.add() handler not added")
			}
		})
	}
}

func Test_routingTree_remove(t *testing.T) {
	type fields struct {
		tree map[string]map[string]http.HandlerFunc
		mu   sync.RWMutex
	}
	type args struct {
		meth string
		path string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{name: "can remove handler if not exists", fields: fields{tree: map[string]map[string]http.HandlerFunc{}, mu: sync.RWMutex{}}, args: args{meth: "GET", path: "/test"}},
		{name: "can remove handler if exists", fields: fields{tree: map[string]map[string]http.HandlerFunc{"GET": {"/test": func(w http.ResponseWriter, r *http.Request) {}}}, mu: sync.RWMutex{}}, args: args{meth: "GET", path: "/test"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RoutingTree{
				tree: tt.fields.tree,
				mu:   tt.fields.mu,
			}
			r.Remove(tt.args.meth, tt.args.path)

			if _, ok := r.tree[tt.args.meth][tt.args.path]; ok {
				t.Errorf("routingTree.remove() handler not removed")
			}
		})
	}
}

func Test_routingTree_get(t *testing.T) {
	type fields struct {
		tree map[string]map[string]http.HandlerFunc
		mu   sync.RWMutex
	}
	type args struct {
		meth string
		path string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   http.HandlerFunc
		want1  bool
	}{
		{name: "Can get handler if exists", fields: fields{tree: map[string]map[string]http.HandlerFunc{"GET": {"/test": func(w http.ResponseWriter, r *http.Request) {}}}, mu: sync.RWMutex{}}, args: args{meth: "GET", path: "/test"}, want: func(w http.ResponseWriter, r *http.Request) {}, want1: true},
		{name: "Return false if path not exists", fields: fields{tree: map[string]map[string]http.HandlerFunc{"GET": {"/test": func(w http.ResponseWriter, r *http.Request) {}}}, mu: sync.RWMutex{}}, args: args{meth: "GET", path: "/test2"}, want: nil, want1: false},
		{name: "Return false if method not exists", fields: fields{tree: map[string]map[string]http.HandlerFunc{"GET": {"/test": func(w http.ResponseWriter, r *http.Request) {}}}, mu: sync.RWMutex{}}, args: args{meth: "POST", path: "/test2"}, want: nil, want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RoutingTree{
				tree: tt.fields.tree,
				mu:   tt.fields.mu,
			}
			handler, got := r.Get(tt.args.meth, tt.args.path)
			if got != tt.want1 {
				t.Errorf("routingTree.get() got1 = %v, want %v", got, tt.want1)
			} else {
				if got {
					if handler == nil {
						t.Errorf("routingTree.get() handler = nil, want %v", tt.want)
					}
				} else {
					if handler != nil {
						t.Errorf("routingTree.get() handler = %v, want nil", handler)
					}
				}
			}
		})
	}
}

func TestNewRoutingTree(t *testing.T) {
	rt := NewRoutingTree()
	if rt.tree == nil {
		t.Errorf("NewRoutingTree() tree not initialized")
	}
}
