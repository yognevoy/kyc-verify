package provider

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kyc-verify/internal/domain"
)

type fakeProvider struct {
	mu    sync.Mutex
	calls int
	errs  []error
}

func (f *fakeProvider) Submit(context.Context, domain.VerificationRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.calls
	f.calls++
	if idx < len(f.errs) {
		return "ref", f.errs[idx]
	}
	return "ref", nil
}

func (f *fakeProvider) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

var errBoom = errors.New("provider boom")

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	const threshold = 2
	const cooldown = 30 * time.Millisecond

	fake := &fakeProvider{errs: []error{nil, errBoom, errBoom, errBoom, nil, nil}}
	cb := NewCircuitBreaker(fake, threshold, cooldown)
	ctx := context.Background()

	steps := []struct {
		name          string
		sleep         time.Duration
		wantErr       error
		wantCallDelta int
	}{
		{name: "closed: success passes through", wantErr: nil, wantCallDelta: 1},
		{name: "closed: first failure counted, still closed", wantErr: errBoom, wantCallDelta: 1},
		{name: "closed: second failure trips breaker open", wantErr: errBoom, wantCallDelta: 1},
		{name: "open: fails fast without calling provider", wantErr: ErrCircuitOpen, wantCallDelta: 0},
		{name: "half-open: trial call fails, reopens", sleep: cooldown * 2, wantErr: errBoom, wantCallDelta: 1},
		{name: "open again: fails fast", wantErr: ErrCircuitOpen, wantCallDelta: 0},
		{name: "half-open: trial call succeeds, closes", sleep: cooldown * 2, wantErr: nil, wantCallDelta: 1},
		{name: "closed: back to normal", wantErr: nil, wantCallDelta: 1},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			if step.sleep > 0 {
				time.Sleep(step.sleep)
			}
			before := fake.callCount()
			_, err := cb.Submit(ctx, domain.VerificationRequest{})
			if !errors.Is(err, step.wantErr) {
				t.Fatalf("Submit() error = %v, want %v", err, step.wantErr)
			}
			if delta := fake.callCount() - before; delta != step.wantCallDelta {
				t.Fatalf("provider called %d times, want %d", delta, step.wantCallDelta)
			}
		})
	}
}

func TestCircuitBreaker_ConcurrentSubmitsDoNotRace(t *testing.T) {
	fake := &fakeProvider{}
	cb := NewCircuitBreaker(fake, 3, 10*time.Millisecond)
	ctx := context.Background()

	var wg sync.WaitGroup
	var successes atomic.Int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := cb.Submit(ctx, domain.VerificationRequest{}); err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()

	if successes.Load() != 100 {
		t.Fatalf("got %d successful submits, want 100 (provider never fails)", successes.Load())
	}
}
