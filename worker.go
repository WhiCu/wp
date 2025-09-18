package wp

import (
	"time"
)

type worker[T any] struct {
	lastUseTime time.Time

	ch chan *WorkerContext[T]

	ctx *WorkerContext[uint32]
}

func NewWorker[T any](ctx *WorkerContext[uint32]) *worker[T] {
	w := worker[T]{
		ch:  make(chan *WorkerContext[T], 1),
		ctx: ctx,
	}

	// TODO: это решение не очень элегантное, стоит пересмотреть
	ctx.AfterFunc(func() { w.ch <- nil })
	return &w
}

func (w *worker[T]) Cancel(cause error) {
	w.ctx.Cancel(cause)
}
