package broadcast

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_Do(t *testing.T) {
	p := createTestPool()

	called := false
	done := make(chan struct{})
	p.Do(func() {
		called = true
		close(done)
	})
	waitOrTimeout(done)

	if !called {
		t.Fatalf("Do did not execute task")
	}
}

func TestPool_Do_WorkerShouldNotExit(t *testing.T) {
	p := createTestPool()

	p.Do(func() {})
	<-time.After(time.Millisecond * 200)

	workers := len(p.tickets)
	if workers == 0 {
		t.Fatalf("worker should not exit after finishing tasks")
	}
}

func TestPool_Do_WorkerShouldExitAfterTimeout(t *testing.T) {
	p := createTestPool()
	p.timeout = time.Millisecond

	p.Do(func() {})
	<-time.After(time.Millisecond * 200)

	workers := len(p.tickets)
	if workers != 0 {
		t.Fatalf("worker should exit after timeout")
	}
}

func TestPool_Do_CapacityReached(t *testing.T) {
	p := createTestPool()
	taskCount := 20
	var workerCount int32 = 10
	p.tickets = make(chan struct{}, workerCount)
	var startedTasks int32 = 0

	for i := 0; i < taskCount; i++ {
		go p.Do(func() {
			atomic.AddInt32(&startedTasks, 1)
			<-time.After(time.Second * 3)
		})
	}

	<-time.After(time.Millisecond * 200)

	if atomic.LoadInt32(&startedTasks) > workerCount {
		t.Fatalf("max worker count should not be exceeded")
	}
}

func TestPool_Do_TaskIsPassedToFreeWorker(t *testing.T) {
	p := createTestPool()
	workerCount := 10
	p.tickets = make(chan struct{}, workerCount)

	for i := 0; i < workerCount; i++ {
		p.Do(func() {})
	}

	called := false
	done := make(chan struct{})
	p.Do(func() {
		called = true
		close(done)
	})
	<-done

	if !called {
		t.Fatalf("queued task was not executed")
	}
}

func createTestPool() *pool {
	return &pool{
		tickets: make(chan struct{}, 1),
		tasks:   make(chan func()),
		timeout: time.Minute * 5,
	}
}
