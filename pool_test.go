package broadcast

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_do(t *testing.T) {
	p := createTestPool()

	called := false
	done := make(chan struct{})
	p.do(func() {
		called = true
		close(done)
	})
	waitOrTimeout(done)

	if !called {
		t.Fatalf("Do did not execute task")
	}
}

func TestPool_do_WorkerShouldNotExit(t *testing.T) {
	p := createTestPool()

	p.do(func() {})
	<-time.After(time.Millisecond * 200)

	workers := len(p.tickets)
	if workers == 0 {
		t.Fatalf("worker should not exit after finishing tasks")
	}
}

func TestPool_do_WorkerShouldExitAfterTimeout(t *testing.T) {
	p := createTestPool()
	p.timeout = time.Millisecond

	p.do(func() {})
	<-time.After(time.Millisecond * 200)

	workers := len(p.tickets)
	if workers != 0 {
		t.Fatalf("worker should exit after timeout")
	}
}

func TestPool_do_CapacityReached(t *testing.T) {
	p := createTestPool()
	taskCount := 20
	var workerCount int32 = 10
	p.tickets = make(chan struct{}, workerCount)
	var startedTasks int32 = 0

	for i := 0; i < taskCount; i++ {
		go p.do(func() {
			atomic.AddInt32(&startedTasks, 1)
			<-time.After(time.Second * 3)
		})
	}

	<-time.After(time.Millisecond * 200)

	if atomic.LoadInt32(&startedTasks) > workerCount {
		t.Fatalf("max worker count should not be exceeded")
	}
}

func TestPool_do_TaskIsPassedToFreeWorker(t *testing.T) {
	p := createTestPool()
	workerCount := 10
	p.tickets = make(chan struct{}, workerCount)

	for i := 0; i < workerCount; i++ {
		p.do(func() {})
	}

	called := false
	done := make(chan struct{})
	p.do(func() {
		called = true
		close(done)
	})
	<-done

	if !called {
		t.Fatalf("queued task was not executed")
	}
}

func TestPool_cancel_ShouldCancelWorkersAndPendingTasks(t *testing.T) {
	p := createTestPool()
	release := make(chan struct{})

	p.do(func() {
		<-release
	})
	pendingCanceled := make(chan struct{})
	go func() {
		p.do(func() {})
		pendingCanceled <- struct{}{}
	}()

	canceledc := make(chan struct{})
	go func() {
		p.cancel()
		close(canceledc)
	}()
	<-pendingCanceled
	close(release)

	select {
	case <-canceledc:
		return
	case <-time.After(time.Second * 3):
		t.Fatalf("cancel didn't force all workers to stop")
	}
}

func createTestPool() *pool {
	return &pool{
		cancelc: make(chan struct{}),
		tickets: make(chan struct{}, 1),
		tasks:   make(chan func()),
		timeout: time.Minute * 5,
	}
}
