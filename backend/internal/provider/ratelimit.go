package provider

import (
	"context"
	"fmt"

	"golang.org/x/time/rate"

	"kyc-verify/internal/domain"
)

type RateLimiter struct {
	next    domain.VerificationProvider
	limiter *rate.Limiter
}

func NewRateLimiter(next domain.VerificationProvider, rps float64, burst int) *RateLimiter {
	return &RateLimiter{next: next, limiter: rate.NewLimiter(rate.Limit(rps), burst)}
}

func (r *RateLimiter) Submit(ctx context.Context, req domain.VerificationRequest) (string, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit wait: %w", err)
	}
	return r.next.Submit(ctx, req)
}
