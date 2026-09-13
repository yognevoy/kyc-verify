package worker

import (
	"context"
	"sync"
)

type Pool[T any] struct {
	jobs    chan T
	process func(context.Context, T)
	wg      sync.WaitGroup
}

func NewPool[T any](queueSize int, process func(context.Context, T)) *Pool[T] {
	return &Pool[T]{
		jobs:    make(chan T, queueSize),
		process: process,
	}
}

func (p *Pool[T]) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.loop(ctx)
	}
}

func (p *Pool[T]) loop(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			p.process(ctx, job)
		case <-ctx.Done():
			return
		}
	}
}

func (p *Pool[T]) Enqueue(ctx context.Context, job T) error {
	select {
	case p.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool[T]) Stop() {
	close(p.jobs)
	p.wg.Wait()
}
