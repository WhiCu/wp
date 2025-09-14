package wp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	wcontext "github.com/WhiCu/wp/context"
	"github.com/WhiCu/wp/pool"
)

type HandlerFunc[T any] = func(c wcontext.Context[T]) error
type ErrHandler = func(err error)

type WorkerPool[T any] struct {
	// Pool
	workerPool *pool.Pool[*worker[T]]

	// Func
	handlerFunc HandlerFunc[T]

	ErrHandler ErrHandler

	ready []*worker[T]

	// Counter
	MaxWorkersCount uint32

	workersCount uint32

	// Time
	MaxIdleWorkerDuration time.Duration

	// Sync
	lock sync.Mutex

	wg sync.WaitGroup

	// Context
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
	var counter atomic.Uint32
	ctx, cancel := context.WithCancelCause(ctx)

	wp.workerPool = pool.New(
		func() *worker[T] {
			ctx := wcontext.New(ctx, counter.Add(1))
			return NewWorker[T](ctx)
		},
	)
	wp.ctx = ctx
	wp.cancel = cancel

	wp.workersCount = 0
	wp.ready = make([]*worker[T], 0, wp.MaxWorkersCount)

	// TODO: add clean
	wp.wg.Go(func() {
		//TODO: добавь функцию которая просто подставляет функцию в select
		log := slog.With("pool", "cleaner")
		var scratch []*worker[T]
		for {
			wp.clean(&scratch)
			log.Info("clean", "count", len(scratch))
			select {
			case <-wp.ctx.Done():
				fmt.Println("cleaner is stopping")
				log.Info("cleaner is stopping")
				return
			default:
				log.Info("cleaner is sleeping")
				time.Sleep(wp.getMaxIdleWorkerDuration())
			}
		}
	})
}

// Serve

func (wp *WorkerPool[T]) Serve(t T) bool {
	w := wp.getWorker()
	if w == nil {
		return false
	}
	// TODO: add timeout
	w.ch <- wcontext.New(context.Background(), t)
	return true
}

func (wp *WorkerPool[T]) getWorker() *worker[T] {
	var w *worker[T]
	createWorker := false

	lock(&wp.lock, func() {
		ready := wp.ready
		n := len(ready) - 1
		if n < 0 {
			if wp.workersCount < wp.MaxWorkersCount {
				createWorker = true
				wp.workersCount++
			}
		} else {
			w = ready[n]
			ready[n] = nil
			wp.ready = ready[:n]
		}
	})

	if w == nil {
		if !createWorker {
			return nil
		}
		w = wp.workerPool.Get()
		wp.wg.Go(func() {
			wp.workerFunc(w)
			wp.workerPool.Put(w)
		})
	}
	return w
}

// Worker

func (wp *WorkerPool[T]) workerFunc(w *worker[T]) {
	defer func() {
		lock(&wp.lock, func() {
			wp.obsolete(w)
		})
	}()
	var c *wcontext.WorkerContext[T]

	// for {
	// 	select {
	// 	case c = <-w.ch:
	// 		wp.handlerFunc(c)
	// 		lock(&wp.lock, func() {
	// 			wp.absolve(w)
	// 		})
	// 	case <-w.ctx.Done():
	// 		if wp.ErrHandler != nil {
	// 			wp.ErrHandler(context.Cause(w.ctx))
	// 		}
	// 		return
	// 	}
	// }

	for c = range w.ch {
		if c == nil {
			if wp.ErrHandler != nil {
				wp.ErrHandler(context.Cause(w.ctx))
			}
			return
		}
		wp.handlerFunc(c)
		lock(&wp.lock, func() {
			wp.absolve(w)
		})
	}
}

func (wp *WorkerPool[T]) obsolete(_ *worker[T]) {
	wp.workersCount--
}

func (wp *WorkerPool[T]) absolve(w *worker[T]) {
	w.lastUseTime = time.Now()
	wp.ready = append(wp.ready, w)
}

func (wp *WorkerPool[T]) Stop() {
	fmt.Println("stop")
	wp.cancel(errors.New("stop"))
	wp.wg.Wait()
	fmt.Println("stopped")
}

// TODO: refactor
func (wp *WorkerPool[T]) clean(scratch *[]*worker[T]) {
	maxIdleWorkerDuration := wp.getMaxIdleWorkerDuration()

	// Clean least recently used workers if they didn't serve connections
	// for more than maxIdleWorkerDuration.
	criticalTime := time.Now().Add(-maxIdleWorkerDuration)

	wp.lock.Lock()
	ready := wp.ready
	n := len(ready)

	// Use binary-search algorithm to find out the index of the least recently worker which can be cleaned up.
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

	// Notify obsolete workers to stop.
	// This notification must be outside the wp.lock, since ch.ch
	// may be blocking and may consume a lot of time if many workers
	// are located on non-local CPUs.
	tmp := *scratch
	for i := range tmp {
		fmt.Println("worker is obsolete", tmp[i].ctx.GetValue())
		//TODO: add err
		tmp[i].Cancel(errors.New("worker is obsolete"))
		tmp[i] = nil
	}
}

func (wp *WorkerPool[T]) getMaxIdleWorkerDuration() time.Duration {
	if wp.MaxIdleWorkerDuration <= 0 {
		return 10 * time.Second
	}
	return wp.MaxIdleWorkerDuration
}

// DEV

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
