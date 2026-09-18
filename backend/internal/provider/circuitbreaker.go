package provider

import (
	"context"
	"errors"
	"sync"
	"time"

	"kyc-verify/internal/domain"
)

var ErrCircuitOpen = errors.New("circuit breaker open: provider unavailable")

type cbState int

const (
	cbClosed cbState = iota
	cbOpen
	cbHalfOpen
)

type CircuitBreaker struct {
	next             domain.VerificationProvider
	failureThreshold int
	cooldown         time.Duration

	mu       sync.Mutex
	state    cbState
	failures int
	openedAt time.Time
}

func NewCircuitBreaker(next domain.VerificationProvider, failureThreshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{next: next, failureThreshold: failureThreshold, cooldown: cooldown}
}

func (c *CircuitBreaker) Submit(ctx context.Context, req domain.VerificationRequest) (string, error) {
	if !c.allow() {
		return "", ErrCircuitOpen
	}
	reference, err := c.next.Submit(ctx, req)
	c.observe(err)
	return reference, err
}

func (c *CircuitBreaker) allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.state {
	case cbOpen:
		if time.Since(c.openedAt) < c.cooldown {
			return false
		}
		c.state = cbHalfOpen
		return true
	case cbHalfOpen:
		return false
	default:
		return true
	}
}

func (c *CircuitBreaker) observe(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		c.failures++
		if c.state == cbHalfOpen || c.failures >= c.failureThreshold {
			c.state = cbOpen
			c.openedAt = time.Now()
		}
		return
	}

	c.failures = 0
	c.state = cbClosed
}
