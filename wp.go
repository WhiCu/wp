package wp

import (
	"sync"
	"time"

	"github.com/WhiCu/wp/pool"
)

type HandlerFunc[T any] func(T) error

type WorkerPool[T any] struct {
	workerPool *pool.Pool[*worker[T]]

	handlerFunc HandlerFunc[T]

	stopCh chan struct{}

	ready []*worker[T]

	MaxWorkersCount int

	MaxIdleWorkerDuration time.Duration

	workersCount int

	lock sync.Mutex

	wg sync.WaitGroup

	mustStop bool
}

func (wp *WorkerPool[T]) Start() {
	if wp.stopCh != nil {
		return
	}
	wp.stopCh = make(chan struct{})
	wp.workerPool = pool.New(
		func() *worker[T] {
			w := &worker[T]{
				ch: make(chan T),
			}
			return w
		},
	)
	// go func() {
	// 	var scratch []*workerChan
	// 	for {
	// 		wp.clean(&scratch)
	// 		select {
	// 		case <-stopCh:
	// 			return
	// 		default:
	// 			time.Sleep(wp.getMaxIdleWorkerDuration())
	// 		}
	// 	}
	// }()
}

func (wp *WorkerPool[T]) getWorker() *worker[T] {
	var ch *worker[T]
	createWorker := false

	wp.lock.Lock()
	ready := wp.ready
	n := len(ready) - 1
	if n < 0 {
		if wp.workersCount < wp.MaxWorkersCount {
			createWorker = true
			wp.workersCount++
		}
	} else {
		ch = ready[n]
		ready[n] = nil
		wp.ready = ready[:n]
	}
	wp.lock.Unlock()

	if ch == nil {
		if !createWorker {
			return nil
		}
		// vch := wp.workerPool.Get()
		// go func() {
		// 	wp.workerFunc(ch)
		// 	wp.workerChanPool.Put(vch)
		// }()
	}
	return ch
}
