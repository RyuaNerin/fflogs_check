package parallel

import (
	"context"
	"sync"
	"time"
)

type Pool interface {
	Reset(ctx context.Context)
	Add(f func(ctx context.Context) error)
	Stop()
	Wait() error
}

type pool struct {
	Pool

	ctx       context.Context
	ctxCancel func()

	wg sync.WaitGroup

	queue     []func(ctx context.Context) error
	queueLock sync.Mutex
	queueWake chan struct{}

	lastError error

	workersLock sync.RWMutex
	workers     int
	workersMax  int
}

func New(workers int) Pool {
	p := &pool{
		queue:      make([]func(ctx context.Context) error, 0, workers),
		queueWake:  make(chan struct{}),
		workersMax: workers,
	}
	p.Reset(context.Background())

	return p
}

func (p *pool) Reset(ctx context.Context) {
	p.ctx, p.ctxCancel = context.WithCancel(ctx)
}

func (p *pool) Add(f func(ctx context.Context) error) {
	p.wg.Add(1)

	p.queueLock.Lock()
	p.queue = append(p.queue, f)
	p.queueLock.Unlock()

	p.workersLock.Lock()
	if p.workers < p.workersMax {
		p.workers++
		go p.work()
	}
	p.workersLock.Unlock()

	select {
	case p.queueWake <- struct{}{}:
	default:
	}
}

func (p *pool) Stop() {
	p.ctxCancel()
}

func (p *pool) Wait() error {
	p.wg.Wait()

	return p.lastError
}

func (p *pool) work() {
	var f func(ctx context.Context) error
	for {
		f = nil

		p.queueLock.Lock()
		if len(p.queue) > 0 {
			f = p.queue[0]
			if len(p.queue) > 1 {
				copy(p.queue, p.queue[1:])
			}
			p.queue = p.queue[:len(p.queue)-1]
		}
		p.queueLock.Unlock()

		if f == nil {
			select {
			case <-time.After(5 * time.Second):
				p.workersLock.Lock()
				p.workers--
				p.workersLock.Unlock()
				return

			case <-p.queueWake:
			}

			continue
		}

		err := f(p.ctx)
		p.wg.Done()
		if err != nil && p.ctx.Err() != nil {
			p.lastError = err
			p.ctxCancel()
		}
	}
}
