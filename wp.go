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
		var scratch []*worker[T]
		for {
			wp.clean(&scratch)
			select {
			case <-wp.ctx.Done():
				return
			default:
				time.Sleep(wp.MaxIdleWorkerDuration)
			}
		}
	})
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
	createWorker := false

	lock(&wp.lock, func() {
		if len(wp.ready) <= 0 {
			if wp.workersCount < wp.MaxWorkersCount {
				createWorker = true
				wp.workersCount++
			}
		} else {
			w = wp.ready.Pop()
		}
	})

	if w == nil {
		if !createWorker {
			return nil
		}
		w = wp.pool.Get()
		wp.wg.Go(func() {
			wp.workerFunc(w)
			wp.pool.Put(w)
		})
	}
	return w
}

func (wp *WorkerPool[T]) workerFunc(w *worker[T]) {
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

	lock(&wp.lock, func() {
		wp.obsolete(w)
	})
}

func (wp *WorkerPool[T]) obsolete(_ *worker[T]) {
	wp.workersCount--
}

func (wp *WorkerPool[T]) absolve(w *worker[T]) {
	w.lastUseTime = time.Now()
	wp.ready.Push(w)
}

func (wp *WorkerPool[T]) Stop() {
	wp.cancel(errors.New("SIGTERM"))
	wp.wg.Wait()
}

func (wp *WorkerPool[T]) clean(scratch *[]*worker[T]) {
	maxIdleWorkerDuration := wp.MaxIdleWorkerDuration

	criticalTime := time.Now().Add(-maxIdleWorkerDuration)

	wp.lock.Lock()
	ready := wp.ready
	n := len(ready)

	l, r := 0, n-1
	for l <= r {
		mid := (l + r) / 2
		if criticalTime.After(wp.ready[mid].lastUseTime) {
			l = mid + 1
		} else {
			r = mid - 1
		}
	}
	i := r
	if i == -1 {
		wp.lock.Unlock()
		return
	}

	*scratch = append((*scratch)[:0], ready[:i+1]...)
	m := copy(ready, ready[i+1:])
	for i = m; i < n; i++ {
		ready[i] = nil
	}
	wp.ready = ready[:m]
	wp.lock.Unlock()

	tmp := *scratch
	for i := range tmp {
		fmt.Println("worker is obsolete", tmp[i].ctx.GetValue())
		// TODO: добавить обработку ошибок
		tmp[i].Cancel(errors.New("worker is obsolete"))
		tmp[i] = nil
	}
}

func lock(lock *sync.Mutex, f func()) {
	lock.Lock()
	defer lock.Unlock()
	f()
}

func (wp *WorkerPool[T]) Status() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("WorkersCount: %d\tMaxWorkersCount: %d", wp.workersCount, wp.MaxWorkersCount))
	return b.String()
}
