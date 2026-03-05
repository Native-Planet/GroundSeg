package workflow

import (
	"context"
	"sync"
)

// AsyncRunHandle exposes completion and terminal error for an asynchronously started worker.
type AsyncRunHandle struct {
	done chan struct{}
	once sync.Once
	mu   sync.RWMutex
	err  error
}

func NewAsyncRunHandle() *AsyncRunHandle {
	return &AsyncRunHandle{
		done: make(chan struct{}),
	}
}

func NewCompletedAsyncRunHandle(err error) *AsyncRunHandle {
	handle := NewAsyncRunHandle()
	handle.Finish(err)
	return handle
}

func (handle *AsyncRunHandle) Done() <-chan struct{} {
	if handle == nil {
		return nil
	}
	return handle.done
}

func (handle *AsyncRunHandle) Err() error {
	if handle == nil {
		return nil
	}
	handle.mu.RLock()
	defer handle.mu.RUnlock()
	return handle.err
}

func (handle *AsyncRunHandle) Finish(err error) {
	if handle == nil {
		return
	}
	handle.once.Do(func() {
		handle.mu.Lock()
		handle.err = err
		handle.mu.Unlock()
		close(handle.done)
	})
}

// StartAsync launches worker in a goroutine and returns an observable handle.
// Nil/canceled contexts produce an already-completed handle.
func StartAsync(ctx context.Context, worker ContextWorker) *AsyncRunHandle {
	if ctx == nil {
		ctx = context.Background()
	}
	handle := NewAsyncRunHandle()
	if ctx.Err() != nil || worker == nil {
		handle.Finish(nil)
		return handle
	}
	go func() {
		handle.Finish(worker(ctx))
	}()
	return handle
}
