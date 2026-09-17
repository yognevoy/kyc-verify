package provider

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"

	"kyc-verify/internal/domain"
)

type ResultCallback func(ctx context.Context, reference string, result domain.VerificationResult)

type MockProvider struct {
	minDelay      time.Duration
	maxDelay      time.Duration
	approveChance float64
	onResult      ResultCallback
}

func NewMockProvider(minDelay, maxDelay time.Duration, approveChance float64, onResult ResultCallback) *MockProvider {
	return &MockProvider{minDelay: minDelay, maxDelay: maxDelay, approveChance: approveChance, onResult: onResult}
}

func (p *MockProvider) Submit(ctx context.Context, _ domain.VerificationRequest) (string, error) {
	reference := uuid.NewString()
	go p.deliver(ctx, reference)
	return reference, nil
}

func (p *MockProvider) deliver(ctx context.Context, reference string) {
	delay := p.minDelay
	if p.maxDelay > p.minDelay {
		delay += time.Duration(rand.Int64N(int64(p.maxDelay - p.minDelay)))
	}

	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return
	}

	result := domain.VerificationResult{Decision: domain.DecisionRejected, Reason: "mock provider: randomly rejected"}
	if rand.Float64() < p.approveChance {
		result = domain.VerificationResult{Decision: domain.DecisionApproved}
	}
	p.onResult(ctx, reference, result)
}
