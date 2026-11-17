package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrQueueFull     = errors.New("job queue is full, please try again in a few seconds")
	ErrTooManyJobs   = errors.New("you already have an active process running, please wait for it to finish")
	ErrJobTimeout    = errors.New("process timed out")
)

type Job func(ctx context.Context) error

type Pool struct {
	maxWorkers          int
	maxProcessesPerUser int
	timeout             time.Duration

	semaphore   chan struct{}
	userCounts  sync.Map // int64 -> *int32
	activeCount int32
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewPool(maxWorkers, maxProcessesPerUser int, timeout time.Duration) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		maxWorkers:          maxWorkers,
		maxProcessesPerUser: maxProcessesPerUser,
		timeout:             timeout,
		semaphore:           make(chan struct{}, maxWorkers),
		ctx:                 ctx,
		cancel:              cancel,
	}
}

func (p *Pool) AcquireUserSlot(userID int64) bool {
	actual, _ := p.userCounts.LoadOrStore(userID, new(int32))
	countPtr := actual.(*int32)

	for {
		cur := atomic.LoadInt32(countPtr)
		if int(cur) >= p.maxProcessesPerUser {
			return false
		}
		if atomic.CompareAndSwapInt32(countPtr, cur, cur+1) {
			return true
		}
	}
}

func (p *Pool) ReleaseUserSlot(userID int64) {
	if actual, ok := p.userCounts.Load(userID); ok {
		countPtr := actual.(*int32)
		atomic.AddInt32(countPtr, -1)
	}
}

func (p *Pool) Submit(userID int64, job Job) error {
	if !p.AcquireUserSlot(userID) {
		return ErrTooManyJobs
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer p.ReleaseUserSlot(userID)

		select {
		case p.semaphore <- struct{}{}:
			defer func() { <-p.semaphore }()
		case <-p.ctx.Done():
			return
		}

		atomic.AddInt32(&p.activeCount, 1)
		defer atomic.AddInt32(&p.activeCount, -1)

		jobCtx, jobCancel := context.WithTimeout(p.ctx, p.timeout)
		defer jobCancel()

		_ = job(jobCtx)
	}()

	return nil
}

func (p *Pool) ActiveWorkers() int {
	return int(atomic.LoadInt32(&p.activeCount))
}

func (p *Pool) Stop() {
	p.cancel()
	p.wg.Wait()
}
