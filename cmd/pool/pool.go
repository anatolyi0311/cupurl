package pool

import (
	"sync"
)

// Pool - обобщенный пул объектов с методом Reset()
type Pool[T interface {
	Reset()
}] struct {
	pool    sync.Pool
	newFunc func() T
}

func New[T interface {
	Reset()
}](newFunc func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
		newFunc: newFunc,
	}
}

func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
