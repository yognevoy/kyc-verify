package domain

import (
	"context"

	"github.com/google/uuid"
)

type VerificationDecision string

const (
	DecisionApproved VerificationDecision = "approved"
	DecisionRejected VerificationDecision = "rejected"
)

type VerificationRequest struct {
	CaseID      uuid.UUID
	ApplicantID uuid.UUID
	Documents   []Document
}

type VerificationResult struct {
	Decision VerificationDecision
	Reason   string
}

type VerificationProvider interface {
	Verify(ctx context.Context, req VerificationRequest) (VerificationResult, error)
}
