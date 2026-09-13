package domain

import (
	"time"

	"github.com/google/uuid"
)

type DocumentType string

const (
	DocumentPassport       DocumentType = "passport"
	DocumentSelfie         DocumentType = "selfie"
	DocumentProofOfAddress DocumentType = "proof_of_address"
)

type Document struct {
	ID          uuid.UUID
	ApplicantID uuid.UUID
	Type        DocumentType
	FilePath    string
	UploadedAt  time.Time
}
