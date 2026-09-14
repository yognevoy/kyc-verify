package domain

import (
	"time"

	"github.com/google/uuid"
)

type ActorType string

const (
	ActorSystem   ActorType = "system"
	ActorUser     ActorType = "user"
	ActorReviewer ActorType = "reviewer"
	ActorProvider ActorType = "provider"
)

type VerificationCaseEvent struct {
	ID         uuid.UUID
	CaseID     uuid.UUID
	FromStatus CaseStatus
	ToStatus   CaseStatus
	ActorType  ActorType
	ActorID    *uuid.UUID
	Comment    *string
	CreatedAt  time.Time
}
