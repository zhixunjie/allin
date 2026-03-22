package engine

import "sync"

// Registry 跟踪所有活跃引擎以支持优雅关闭。
type Registry struct {
	mu      sync.Mutex           // 保护 engines map 的并发访问
	engines map[*Engine]struct{} // 活跃引擎集合
	wg      sync.WaitGroup       // 等待所有引擎退出
}

// NewRegistry 创建一个空的引擎注册表。
func NewRegistry() *Registry {
	return &Registry{engines: make(map[*Engine]struct{})}
}

// track 在引擎启动前注册它。
func (r *Registry) track(e *Engine) {
	r.mu.Lock()
	r.engines[e] = struct{}{}
	r.mu.Unlock()
	r.wg.Add(1)
}

// done 在引擎退出后移除它。
func (r *Registry) done(e *Engine) {
	r.mu.Lock()
	delete(r.engines, e)
	r.mu.Unlock()
	r.wg.Done()
}

// StopAll 通知所有运行中的引擎停止并阻塞直到全部退出。
func (r *Registry) StopAll() {
	r.mu.Lock()
	for e := range r.engines {
		e.Stop()
	}
	r.mu.Unlock()
	r.wg.Wait()
}
