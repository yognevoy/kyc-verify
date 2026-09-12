package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Applicant struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FullName  string
	BirthDate time.Time
	Country   string
	RiskLevel RiskLevel
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrApplicantNotFound      = errors.New("applicant not found")
	ErrApplicantAlreadyExists = errors.New("applicant already exists")
)
