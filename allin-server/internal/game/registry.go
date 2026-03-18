package game

import "sync"

// Registry tracks all active engines for graceful shutdown.
type Registry struct {
	mu      sync.Mutex
	engines map[*Engine]struct{}
	wg      sync.WaitGroup
}

// NewRegistry creates an empty engine registry.
func NewRegistry() *Registry {
	return &Registry{engines: make(map[*Engine]struct{})}
}

// track registers an engine before starting it.
func (r *Registry) track(e *Engine) {
	r.mu.Lock()
	r.engines[e] = struct{}{}
	r.mu.Unlock()
	r.wg.Add(1)
}

// done removes an engine after it exits.
func (r *Registry) done(e *Engine) {
	r.mu.Lock()
	delete(r.engines, e)
	r.mu.Unlock()
	r.wg.Done()
}

// StopAll signals every running engine to stop and blocks until all exit.
func (r *Registry) StopAll() {
	r.mu.Lock()
	for e := range r.engines {
		e.Stop()
	}
	r.mu.Unlock()
	r.wg.Wait()
}
