package broadcast

import "time"

const defaultPoolSize int32 = 1000
const defaultPoolTimeout time.Duration = time.Minute * 5

type pool struct {
	tickets chan struct{}
	tasks   chan func()
	timeout time.Duration
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
		}
	}
}

func (p *pool) Do(task func()) {
	select {
	case p.tasks <- task:
	case p.tickets <- struct{}{}:
		go func() {
			p.worker(task)
			<-p.tickets
		}()
	}
}
