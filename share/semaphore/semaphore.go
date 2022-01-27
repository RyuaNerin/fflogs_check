package semaphore

type Semaphore struct {
	ch chan struct{}
}

func New(max int) *Semaphore {
	sema := &Semaphore{
		ch: make(chan struct{}, max),
	}
	for i := 0; i < max; i++ {
		sema.ch <- struct{}{}
	}

	return sema
}

func (sema *Semaphore) Acquire() {
	<-sema.ch
}

func (sema *Semaphore) Release() {
	sema.ch <- struct{}{}
}
