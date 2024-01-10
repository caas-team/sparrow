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
)

// creates a simple routing tree, so checks can easily create and remove handlers
// Maps the method to the path and the handler
type RoutingTree struct {
	tree map[string]map[string]http.HandlerFunc
	mu   sync.RWMutex
}

func (r *RoutingTree) Add(method, path string, handler http.HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tree[method]; !ok {
		r.tree[method] = make(map[string]http.HandlerFunc)
	}
	r.tree[method][path] = handler
}

func (r *RoutingTree) Remove(meth, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tree[meth]; !ok {
		return
	}
	delete(r.tree[meth], path)
}

func (r *RoutingTree) Get(method, path string) (http.HandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.tree[method]; !ok {
		return nil, false
	}
	handler, ok := r.tree[method][path]
	return handler, ok
		
}

func NewRoutingTree() *RoutingTree {
	return &RoutingTree{
		tree: make(map[string]map[string]http.HandlerFunc),
		mu:   sync.RWMutex{},
	}
}
