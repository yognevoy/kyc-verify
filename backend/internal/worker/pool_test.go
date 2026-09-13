package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_ProcessesAllJobsConcurrentlyWithoutRaces(t *testing.T) {
	const jobCount = 500
	const workerCount = 8

	var processed atomic.Int64
	var mu sync.Mutex
	seen := make(map[int]bool, jobCount)

	pool := NewPool[int](jobCount, func(_ context.Context, job int) {
		processed.Add(1)
		mu.Lock()
		seen[job] = true
		mu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx, workerCount)

	var wg sync.WaitGroup
	for i := 0; i < jobCount; i++ {
		wg.Add(1)
		go func(job int) {
			defer wg.Done()
			if err := pool.Enqueue(ctx, job); err != nil {
				t.Errorf("enqueue job %d: %v", job, err)
			}
		}(i)
	}
	wg.Wait()

	pool.Stop()

	if got := processed.Load(); got != jobCount {
		t.Fatalf("processed %d jobs, want %d", got, jobCount)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != jobCount {
		t.Fatalf("saw %d distinct jobs, want %d", len(seen), jobCount)
	}
}

func TestPool_StopWaitsForInFlightJob(t *testing.T) {
	var done atomic.Bool

	pool := NewPool[int](1, func(_ context.Context, _ int) {
		time.Sleep(50 * time.Millisecond)
		done.Store(true)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx, 1)
	if err := pool.Enqueue(ctx, 1); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	pool.Stop()

	if !done.Load() {
		t.Fatal("Stop returned before the in-flight job finished")
	}
}

func TestPool_EnqueueRespectsContextCancellation(t *testing.T) {
	pool := NewPool[int](0, func(context.Context, int) {})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := pool.Enqueue(ctx, 1); err == nil {
		t.Fatal("expected Enqueue to fail on an already-cancelled context")
	}
}
