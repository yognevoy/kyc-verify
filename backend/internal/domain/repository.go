package domain

import (
	"context"
	"io"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type ApplicantRepository interface {
	Create(ctx context.Context, applicant *Applicant) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Applicant, error)
	Update(ctx context.Context, applicant *Applicant) error
}

type DocumentRepository interface {
	Create(ctx context.Context, doc *Document) error
	ListByApplicantID(ctx context.Context, applicantID uuid.UUID) ([]Document, error)
}

type DocumentStorage interface {
	Save(ctx context.Context, applicantID uuid.UUID, docType DocumentType, filename string, r io.Reader) (path string, err error)
}

type VerificationCaseRepository interface {
	Create(ctx context.Context, c *VerificationCase) error
	GetByID(ctx context.Context, id uuid.UUID) (*VerificationCase, error)
	GetLatestByApplicantID(ctx context.Context, applicantID uuid.UUID) (*VerificationCase, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status CaseStatus) error
}
