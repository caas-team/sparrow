// sparrow
// (C) 2023, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package api

import (
	"net/http"
	"sync"
	"testing"
)

func Test_routingTree_add(t *testing.T) {
	type args struct {
		meth    string
		path    string
		handler http.HandlerFunc
	}
	tests := []struct {
		name  string
		rtree *RoutingTree
		args  args
	}{
		{name: "can add handler", rtree: &RoutingTree{tree: map[string]map[string]http.HandlerFunc{}, mu: sync.RWMutex{}}, args: args{meth: http.MethodGet, path: "/test", handler: func(w http.ResponseWriter, r *http.Request) {}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rtree.Add(tt.args.meth, tt.args.path, tt.args.handler)

			if _, ok := tt.rtree.tree[tt.args.meth][tt.args.path]; !ok {
				t.Errorf("routingTree.add() handler not added")
			}
		})
	}
}

func Test_routingTree_remove(t *testing.T) {
	type args struct {
		meth string
		path string
	}
	tests := []struct {
		name  string
		rtree *RoutingTree
		args  args
	}{
		{name: "can remove handler if not exists", rtree: &RoutingTree{tree: map[string]map[string]http.HandlerFunc{}, mu: sync.RWMutex{}}, args: args{meth: http.MethodGet, path: "/test"}},
		{name: "can remove handler if exists", rtree: &RoutingTree{tree: map[string]map[string]http.HandlerFunc{http.MethodGet: {"/test": func(w http.ResponseWriter, r *http.Request) {}}}, mu: sync.RWMutex{}}, args: args{meth: http.MethodGet, path: "/test"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rtree.Remove(tt.args.meth, tt.args.path)

			if _, ok := tt.rtree.tree[tt.args.meth][tt.args.path]; ok {
				t.Errorf("routingTree.remove() handler not removed")
			}
		})
	}
}

func Test_routingTree_get(t *testing.T) {
	type args struct {
		meth string
		path string
	}
	tests := []struct {
		name  string
		rtree *RoutingTree
		args  args
		want  http.HandlerFunc
		want1 bool
	}{
		{name: "Can get handler if exists", rtree: &RoutingTree{tree: map[string]map[string]http.HandlerFunc{http.MethodGet: {"/test": func(w http.ResponseWriter, r *http.Request) {}}}, mu: sync.RWMutex{}}, args: args{meth: http.MethodGet, path: "/test"}, want: func(w http.ResponseWriter, r *http.Request) {}, want1: true},
		{name: "Return false if path not exists", rtree: &RoutingTree{tree: map[string]map[string]http.HandlerFunc{http.MethodGet: {"/test": func(w http.ResponseWriter, r *http.Request) {}}}, mu: sync.RWMutex{}}, args: args{meth: http.MethodGet, path: "/test2"}, want: nil, want1: false},
		{name: "Return false if method not exists", rtree: &RoutingTree{tree: map[string]map[string]http.HandlerFunc{http.MethodGet: {"/test": func(w http.ResponseWriter, r *http.Request) {}}}, mu: sync.RWMutex{}}, args: args{meth: http.MethodPost, path: "/test2"}, want: nil, want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, got := tt.rtree.Get(tt.args.meth, tt.args.path)
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
		t.Errorf("NewRoutingTree() rtree not initialized")
	}
}
