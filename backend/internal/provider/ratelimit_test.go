package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"kyc-verify/internal/domain"
)

func TestRateLimiter_ThrottlesBeyondBurst(t *testing.T) {
	fake := &fakeProvider{}
	rl := NewRateLimiter(fake, 10, 1)
	ctx := context.Background()

	start := time.Now()
	if _, err := rl.Submit(ctx, domain.VerificationRequest{}); err != nil {
		t.Fatalf("first Submit: %v", err)
	}
	firstElapsed := time.Since(start)
	if firstElapsed > 20*time.Millisecond {
		t.Fatalf("first call waited %v, want near-instant (burst token available)", firstElapsed)
	}

	start = time.Now()
	if _, err := rl.Submit(ctx, domain.VerificationRequest{}); err != nil {
		t.Fatalf("second Submit: %v", err)
	}
	secondElapsed := time.Since(start)
	if secondElapsed < 50*time.Millisecond {
		t.Fatalf("second call waited %v, want it throttled by the limiter", secondElapsed)
	}

	if got := fake.callCount(); got != 2 {
		t.Fatalf("provider called %d times, want 2", got)
	}
}

func TestRateLimiter_RespectsContextCancellation(t *testing.T) {
	fake := &fakeProvider{}
	rl := NewRateLimiter(fake, 1, 1)
	ctx, cancel := context.WithCancel(context.Background())

	if _, err := rl.Submit(ctx, domain.VerificationRequest{}); err != nil {
		t.Fatalf("first Submit: %v", err)
	}
	cancel()

	if _, err := rl.Submit(ctx, domain.VerificationRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("second Submit() error = %v, want context.Canceled", err)
	}
	if got := fake.callCount(); got != 1 {
		t.Fatalf("provider called %d times, want 1 (second call should never reach it)", got)
	}
}
