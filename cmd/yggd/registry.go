package main

import (
	"sync"
)

var errWorkerRegistered = registryError("cannot add worker; a worker is already registered")

type registryError string

func (e registryError) Error() string { return string(e) }

type registry struct {
	mu sync.RWMutex
	mp map[string]*workerConfig
}

func (r *registry) init() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.mp == nil {
		r.mp = make(map[string]*workerConfig)
	}
}

func (r *registry) set(directive string, w *workerConfig) error {
	r.init()

	if r.get(directive) != nil {
		return errWorkerRegistered
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.mp[directive] = w

	return nil
}

func (r *registry) get(directive string) *workerConfig {
	r.init()

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.mp[directive]
}

func (r *registry) del(directive string) {
	r.init()

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.mp, directive)
}

func (r *registry) all() map[string]*workerConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := make(map[string]*workerConfig)
	for k, w := range r.mp {
		all[k] = w
	}

	return all
}
