package parallel

import (
	"context"
	"sync"
)

type Pool interface {
	Reset(ctx context.Context)
	Do(f func(ctx context.Context) error)
	Abort()
	Wait() error
}

type pool struct {
	Pool

	ctx       context.Context
	ctxCancel func()

	wg sync.WaitGroup

	workers chan struct{}

	lastErrorLock sync.Mutex
	lastError     error
}

func New(workers int) Pool {
	p := &pool{
		workers: make(chan struct{}, workers),
	}
	p.Reset(context.Background())

	for i := 0; i < workers; i++ {
		p.workers <- struct{}{}
	}

	return p
}

func (p *pool) Reset(ctx context.Context) {
	p.ctx, p.ctxCancel = context.WithCancel(ctx)
}

func (p *pool) Do(f func(ctx context.Context) error) {
	p.wg.Add(1)
	<-p.workers

	go func() {
		defer func() {
			p.workers <- struct{}{}
			p.wg.Done()
		}()

		if p.ctx.Err() != nil {
			return
		}

		err := f(p.ctx)
		if err != nil {
			p.lastErrorLock.Lock()
			p.lastError = err
			p.lastErrorLock.Unlock()
		}

	}()
}

func (p *pool) Abort() {
	p.ctxCancel()
}

func (p *pool) Wait() error {
	p.wg.Wait()

	return p.lastError
}
