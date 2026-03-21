package async

import "sync"

// Manager manages background goroutines and ensures they complete before exit.
type Manager struct {
	wg sync.WaitGroup
}

// defaultManager is the global instance used by package-level functions.
var defaultManager = &Manager{}

// Go spawns a goroutine and tracks it for graceful shutdown.
// The function f is executed in a new goroutine.
func (m *Manager) Go(f func()) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		f()
	}()
}

// Wait blocks until all tracked goroutines have completed.
func (m *Manager) Wait() {
	m.wg.Wait()
}

// Go spawns a goroutine using the default manager.
func Go(f func()) {
	defaultManager.Go(f)
}

// Wait blocks until all goroutines spawned via the default manager have completed.
func Wait() {
	defaultManager.Wait()
}
