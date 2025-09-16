package wp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WhiCu/wp/pool"
)

type HandlerFunc[T any] = func(c Context[T])

type ErrHandler = HandlerFunc[error]

type WorkerPool[T any] struct {
	pool *pool.Pool[*worker[T]]

	handlerFunc HandlerFunc[T]

	ErrHandler ErrHandler

	ready stack[*worker[T]]

	MaxWorkersCount uint32

	workersCount uint32

	MinWorkersCount uint32

	MaxIdleWorkerDuration time.Duration

	lock sync.Mutex

	wg sync.WaitGroup

	ctx context.Context

	cancel context.CancelCauseFunc
}

func New[T any](handlerFunc HandlerFunc[T], opts ...Option[T]) *WorkerPool[T] {
	wp := &WorkerPool[T]{
		handlerFunc: handlerFunc,
	}

	for _, opt := range opts {
		opt(wp)
	}

	return wp
}

func (wp *WorkerPool[T]) Start(ctx context.Context) {
	wp.ctx, wp.cancel = context.WithCancelCause(ctx)

	var counter atomic.Uint32

	wp.pool = pool.New(
		func() *worker[T] {
			ctx := NewContext(wp.ctx, counter.Add(1))
			return NewWorker[T](ctx)
		},
	)

	wp.workersCount = 0
	wp.ready = make([]*worker[T], 0, wp.MaxWorkersCount)

	wp.wg.Go(func() {
		for {
			wp.clean()
			select {
			case <-wp.ctx.Done():
				return
			default:
				time.Sleep(wp.MaxIdleWorkerDuration)
			}
		}
	})

	var w *worker[T]
	for range wp.MinWorkersCount {
		w = wp.createWorker()
		wp.ready.Push(w)
	}
}

func (wp *WorkerPool[T]) Serve(t T) bool {

	w := wp.getWorker()
	if w == nil {
		return false
	}

	// TODO: добавить таймаут для отправки задачи, а ещё нужно переработать worker'ов, задачу можно завершить из вне
	w.ch <- NewContext(w.ctx, t)
	return true
}

func (wp *WorkerPool[T]) getWorker() *worker[T] {
	var w *worker[T]

	lock(&wp.lock, func() {
		if len(wp.ready) <= 0 {
			if wp.workersCount < wp.MaxWorkersCount {
				w = wp.createWorker()
			}
		} else {
			w = wp.ready.Pop()
		}
	})
	return w
}

func (wp *WorkerPool[T]) createWorker() (w *worker[T]) {
	// TODO: наверное было бы хорошо увидеть здесь lock
	w = wp.pool.Get()
	w.obsoleted = false
	wp.wg.Go(func() {
		wp.workerFunc(w)
		wp.pool.Put(w)
	})
	wp.workersCount++
	return w
}

func (wp *WorkerPool[T]) workerFunc(w *worker[T]) {
	defer lock(&wp.lock, func() {
		wp.obsolete(w)
	})
	var c *WorkerContext[T]

	for c = range w.ch {
		if c == nil {
			if wp.ErrHandler != nil {
				wp.ErrHandler(NewContext(wp.ctx, context.Cause(w.ctx)))
			}
			break
		}
		wp.handlerFunc(c)

		lock(&wp.lock, func() {
			wp.absolve(w)
		})
	}

}

func (wp *WorkerPool[T]) obsolete(w *worker[T]) {
	wp.workersCount--
	w.obsoleted = true
}

func (wp *WorkerPool[T]) absolve(w *worker[T]) {
	w.lastUseTime = time.Now()
	wp.ready.Push(w)
}

func (wp *WorkerPool[T]) Stop() {
	wp.cancel(errors.New("SIGTERM"))
	wp.wg.Wait()
}

func (wp *WorkerPool[T]) clean() {
	// TODO: убивает всех до нуля а не MinWorkersCount если проходит условие
	lock(&wp.lock, func() {
		if wp.workersCount <= wp.MinWorkersCount {
			return
		}

		maxIdleWorkerDuration := wp.MaxIdleWorkerDuration

		criticalTime := time.Now().Add(-maxIdleWorkerDuration)

		var w *worker[T]
		var i int

		for i, w = range wp.ready {
			if len(wp.ready)-i <= int(wp.MinWorkersCount) {
				break
			}
			if criticalTime.Before(w.lastUseTime) {
				break
			}
			w.obsoleted = true
			w.Cancel(errors.New("idle worker"))
		}

		wp.ready = wp.ready[i:]

		// TODO: скорее всего бесполезно
		for i, w = range wp.ready {
			if w.obsoleted {
				wp.ready = append(wp.ready[:i], wp.ready[i+1:]...)
			}
		}
	})

}

func lock(lock *sync.Mutex, f func()) {
	lock.Lock()
	defer lock.Unlock()
	f()
}

func (wp *WorkerPool[T]) Status() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("-WorkersCount: %d\t↑MaxWorkersCount: %d\t↓MinWorkersCount: %d", wp.workersCount, wp.MaxWorkersCount, wp.MinWorkersCount))
	return b.String()
}
