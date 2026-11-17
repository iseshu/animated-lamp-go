package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolConcurrencyLimit(t *testing.T) {
	maxWorkers := 2
	pool := NewPool(maxWorkers, 5, 2*time.Second)
	defer pool.Stop()

	var running int32
	var maxObserved int32

	for i := 0; i < 6; i++ {
		err := pool.Submit(int64(i+1), func(ctx context.Context) error {
			current := atomic.AddInt32(&running, 1)
			defer atomic.AddInt32(&running, -1)

			for {
				old := atomic.LoadInt32(&maxObserved)
				if current <= old || atomic.CompareAndSwapInt32(&maxObserved, old, current) {
					break
				}
			}

			time.Sleep(50 * time.Millisecond)
			return nil
		})
		if err != nil {
			t.Fatalf("failed to submit job %d: %v", i, err)
		}
	}

	time.Sleep(300 * time.Millisecond)

	observed := atomic.LoadInt32(&maxObserved)
	if int(observed) > maxWorkers {
		t.Errorf("max concurrent observed = %d, wanted <= %d", observed, maxWorkers)
	}
}

func TestPoolPerUserLimit(t *testing.T) {
	pool := NewPool(5, 1, 1*time.Second)
	defer pool.Stop()

	userID := int64(12345)

	// First job should succeed
	err := pool.Submit(userID, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("first job failed: %v", err)
	}

	// Second job for same user while first is active should fail with ErrTooManyJobs
	err = pool.Submit(userID, func(ctx context.Context) error {
		return nil
	})
	if err != ErrTooManyJobs {
		t.Fatalf("expected ErrTooManyJobs, got: %v", err)
	}
}
