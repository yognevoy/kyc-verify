package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CaseStatus string

const (
	StatusDraft     CaseStatus = "draft"
	StatusSubmitted CaseStatus = "submitted"
	StatusInReview  CaseStatus = "in_review"
	StatusApproved  CaseStatus = "approved"
	StatusRejected  CaseStatus = "rejected"
)

type VerificationCase struct {
	ID          uuid.UUID
	ApplicantID uuid.UUID
	Status      CaseStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var (
	ErrVerificationCaseNotFound = errors.New("verification case not found")
	ErrInvalidTransition        = errors.New("invalid verification case status transition")
)

var allowedTransitions = map[CaseStatus][]CaseStatus{
	StatusDraft:     {StatusSubmitted},
	StatusSubmitted: {StatusInReview},
	StatusInReview:  {StatusApproved, StatusRejected},
}

func (c *VerificationCase) TransitionTo(status CaseStatus) error {
	for _, allowed := range allowedTransitions[c.Status] {
		if allowed == status {
			c.Status = status
			return nil
		}
	}
	return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, c.Status, status)
}

type QueueItem struct {
	CaseID            uuid.UUID
	ApplicantID       uuid.UUID
	ApplicantFullName string
	ApplicantRisk     RiskLevel
	PriorRejections   int
	Status            CaseStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
