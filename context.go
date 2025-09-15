package wp

import (
	"context"
	"time"
)

type Context[T any] interface {
	context.Context
	// Cancel(cause error)
	GetValue() T
}

type WorkerContext[T any] struct {
	ctx context.Context

	cancel context.CancelCauseFunc

	value T
}

var (
	_ context.Context = &WorkerContext[any]{}

	_ Context[any] = &WorkerContext[any]{}
)

func NewContext[T any](ctx context.Context, value T) *WorkerContext[T] {
	ctx, cancel := context.WithCancelCause(ctx)
	return &WorkerContext[T]{
		ctx:    ctx,
		cancel: cancel,
		value:  value,
	}
}

func (c *WorkerContext[T]) AfterFunc(f func()) {
	context.AfterFunc(c.ctx, f)
}

func (c *WorkerContext[T]) Cancel(cause error) {
	c.cancel(cause)
}

func (c *WorkerContext[T]) Context() context.Context {
	if c == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *WorkerContext[T]) Deadline() (deadline time.Time, ok bool) {
	return c.Context().Deadline()
}

func (c *WorkerContext[T]) Done() <-chan struct{} {
	return c.Context().Done()
}

func (c *WorkerContext[T]) Err() error {
	return c.Context().Err()
}

func (c *WorkerContext[T]) Value(key any) any {
	return c.Context().Value(key)
}

func (c *WorkerContext[T]) GetValue() T {
	return c.value
}
