package wp

import (
	"time"

	"github.com/WhiCu/wp/context"
)

type worker[T any] struct {
	lastUseTime time.Time
	ch          chan *context.WorkerContext[T]

	// ctx
	ctx *context.WorkerContext[uint32]
}

func NewWorker[T any](ctx *context.WorkerContext[uint32]) *worker[T] {
	return &worker[T]{
		ch:  make(chan *context.WorkerContext[T], 1),
		ctx: ctx,
	}
}

func (w *worker[T]) Cancel(cause error) {
	w.ctx.Cancel(cause)
}
