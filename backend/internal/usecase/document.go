package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"kyc-verify/internal/domain"
)

var ErrInvalidDocumentType = errors.New("invalid document type")

var allowedDocumentTypes = map[domain.DocumentType]bool{
	domain.DocumentPassport:       true,
	domain.DocumentSelfie:         true,
	domain.DocumentProofOfAddress: true,
}

type DocumentUsecase struct {
	documents domain.DocumentRepository
	storage   domain.DocumentStorage
}

func NewDocumentUsecase(documents domain.DocumentRepository, storage domain.DocumentStorage) *DocumentUsecase {
	return &DocumentUsecase{documents: documents, storage: storage}
}

func (u *DocumentUsecase) Upload(ctx context.Context, applicantID uuid.UUID, docType domain.DocumentType, filename string, r io.Reader) (*domain.Document, error) {
	if !allowedDocumentTypes[docType] {
		return nil, ErrInvalidDocumentType
	}

	path, err := u.storage.Save(ctx, applicantID, docType, filename, r)
	if err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	doc := &domain.Document{
		ID:          uuid.New(),
		ApplicantID: applicantID,
		Type:        docType,
		FilePath:    path,
	}
	if err := u.documents.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("create document record: %w", err)
	}
	return doc, nil
}

func (u *DocumentUsecase) ListByApplicantID(ctx context.Context, applicantID uuid.UUID) ([]domain.Document, error) {
	return u.documents.ListByApplicantID(ctx, applicantID)
}
