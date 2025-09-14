package wp

import "time"

type Option[T any] func(wp *WorkerPool[T])

func WithMaxWorkersCount[T any](count uint32) Option[T] {
	return func(wp *WorkerPool[T]) {
		wp.MaxWorkersCount = count
	}
}

func WithMaxIdleWorkerDuration[T any](duration time.Duration) Option[T] {
	return func(wp *WorkerPool[T]) {
		wp.MaxIdleWorkerDuration = duration
	}
}

func WithErrHandler[T any](handler ErrHandler) Option[T] {
	return func(wp *WorkerPool[T]) {
		wp.ErrHandler = handler
	}
}
