package provider

import (
	"context"
	"math/rand/v2"
	"time"

	"kyc-verify/internal/domain"
)

type MockProvider struct {
	minDelay      time.Duration
	maxDelay      time.Duration
	approveChance float64
}

func NewMockProvider(minDelay, maxDelay time.Duration, approveChance float64) *MockProvider {
	return &MockProvider{minDelay: minDelay, maxDelay: maxDelay, approveChance: approveChance}
}

func (p *MockProvider) Verify(ctx context.Context, _ domain.VerificationRequest) (domain.VerificationResult, error) {
	delay := p.minDelay
	if p.maxDelay > p.minDelay {
		delay += time.Duration(rand.Int64N(int64(p.maxDelay - p.minDelay)))
	}

	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return domain.VerificationResult{}, ctx.Err()
	}

	if rand.Float64() < p.approveChance {
		return domain.VerificationResult{Decision: domain.DecisionApproved}, nil
	}
	return domain.VerificationResult{Decision: domain.DecisionRejected, Reason: "mock provider: randomly rejected"}, nil
}
