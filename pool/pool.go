package pool

import "sync"

type Pool[T any] struct {
	sync.Pool
}

func New[T any](f func() T) *Pool[T] {
	return &Pool[T]{
		Pool: sync.Pool{
			New: func() any {
				return f()
			},
		},
	}
}

func (p *Pool[T]) Get() T {
	return p.Pool.Get().(T)
}

func (p *Pool[T]) Put(t T) {
	p.Pool.Put(t)
}
