package wp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	wcontext "github.com/WhiCu/wp/context"
	"github.com/WhiCu/wp/pool"
)

type HandlerFunc[T any] = func(c wcontext.Context[T]) error

type WorkerPool[T any] struct {
	// Pool
	workerPool *pool.Pool[*worker[T]]

	// Func
	handlerFunc HandlerFunc[T]

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

	cancel context.CancelFunc
}

func New[T any](handlerFunc HandlerFunc[T], ctx context.Context) *WorkerPool[T] {
	ctx, cancel := context.WithCancel(ctx)
	var counter atomic.Uint32
	return &WorkerPool[T]{
		handlerFunc: handlerFunc,

		workerPool: pool.New(
			func() *worker[T] {
				ctx := wcontext.New(ctx, counter.Add(1))
				return NewWorker[T](ctx)
			},
		),

		ctx:    ctx,
		cancel: cancel,
	}
}

func (wp *WorkerPool[T]) Start() {

	// TODO: add clean
	go func() {
		var scratch []*worker[T]
		for {
			wp.clean(&scratch)
			select {
			case <-wp.ctx.Done():
				return
			default:
				time.Sleep(wp.getMaxIdleWorkerDuration())
			}
		}
	}()
}

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

func (wp *WorkerPool[T]) workerFunc(w *worker[T]) {
	var c *wcontext.WorkerContext[T]

	for c = range w.ch {
		wp.handlerFunc(c)
		lock(&wp.lock, func() {
			w.lastUseTime = time.Now()
			wp.ready = append(wp.ready, w)
		})
		select {
		case <-w.ctx.Done():
			lock(&wp.lock, func() {
				wp.workersCount--
			})
			return
		default:
		}
	}
}

func (wp *WorkerPool[T]) Stop() {
	wp.cancel()
	wp.wg.Wait()
}

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

//

func lock(lock *sync.Mutex, f func()) {
	lock.Lock()
	defer lock.Unlock()
	f()
}

func (wp *WorkerPool[T]) Status() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("WorkersCount: %d\tMaxWorkersCount: %d\n", wp.workersCount, wp.MaxWorkersCount))
	return b.String()
}
