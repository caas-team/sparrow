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

func NewRoutingTree() RoutingTree {
	return RoutingTree{
		tree: make(map[string]map[string]http.HandlerFunc),
		mu:   sync.RWMutex{},
	}
}
