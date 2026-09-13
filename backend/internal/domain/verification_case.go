package domain

import (
	"errors"
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

var ErrVerificationCaseNotFound = errors.New("verification case not found")
