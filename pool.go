package broadcast

import (
	"time"
)

const defaultPoolSize int32 = 100
const defaultPoolTimeout time.Duration = time.Minute * 5

type pool struct {
	cancelc chan struct{}
	tickets chan struct{}
	tasks   chan func()
	timeout time.Duration
}

func (p *pool) cancel() {
	close(p.cancelc)
	cap := cap(p.tickets)

	// Wait for all pool go routines to exit.
	for i := 0; i < cap; i++ {
		p.tickets <- struct{}{}
	}
}

func (p *pool) worker(task func()) {
	task()
	timeout := time.After(p.timeout)

	for {
		select {
		case t := <-p.tasks:
			t()
		case <-timeout:
			return
		case <-p.cancelc:
			return
		}
	}
}

func (p *pool) do(task func()) {
	select {
	case <-p.cancelc:
		return
	case p.tasks <- task:
	case p.tickets <- struct{}{}:
		go func() {
			p.worker(task)
			<-p.tickets
		}()
	}
}
