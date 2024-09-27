package fn

import (
	"context"
	"errors"
	"sync"
)

// ErrStopping is returned when trying to add a new goroutine while stopping.
var ErrStopping = errors.New("can not add goroutine, stopping")

// GoroutineManager is used to launch goroutines until context expires or the
// manager is stopped. The Stop method blocks until all started goroutines stop.
type GoroutineManager struct {
	wg     sync.WaitGroup
	mu     sync.Mutex
	ctx    context.Context
	cancel func()
}

// NewGoroutineManager constructs and returns a new instance of
// GoroutineManager.
func NewGoroutineManager(ctx context.Context) *GoroutineManager {
	ctx, cancel := context.WithCancel(ctx)

	return &GoroutineManager{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Go starts a new goroutine if the manager is not stopping.
func (g *GoroutineManager) Go(f func(ctx context.Context)) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.ctx.Err() != nil {
		return ErrStopping
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		f(g.ctx)
	}()

	return nil
}

// Stop prevents new goroutines from being added and waits for all running
// goroutines to finish.
func (g *GoroutineManager) Stop() {
	g.mu.Lock()
	g.cancel()
	g.mu.Unlock()

	// Wait for all goroutines to finish.
	g.wg.Wait()
}

// Context returns internal context of the GoroutineManager which will expire
// when either the context passed to NewGoroutineManager expires or when Stop
// is called.
func (g *GoroutineManager) Context() context.Context {
	return g.ctx
}
