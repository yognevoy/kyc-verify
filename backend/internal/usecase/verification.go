package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"

	"kyc-verify/internal/domain"
)

var (
	ErrMissingDocuments   = errors.New("passport, selfie and proof of address are all required")
	ErrCaseAlreadyPending = errors.New("a verification case is already in progress")
)

var requiredDocumentTypes = [...]domain.DocumentType{
	domain.DocumentPassport,
	domain.DocumentSelfie,
	domain.DocumentProofOfAddress,
}

type CaseQueue interface {
	Enqueue(ctx context.Context, caseID uuid.UUID) error
}

type VerificationUsecase struct {
	cases     domain.VerificationCaseRepository
	documents domain.DocumentRepository
	provider  domain.VerificationProvider
	queue     CaseQueue
}

func NewVerificationUsecase(
	cases domain.VerificationCaseRepository,
	documents domain.DocumentRepository,
	provider domain.VerificationProvider,
	queue CaseQueue,
) *VerificationUsecase {
	return &VerificationUsecase{cases: cases, documents: documents, provider: provider, queue: queue}
}

func (u *VerificationUsecase) Submit(ctx context.Context, applicantID uuid.UUID) (*domain.VerificationCase, error) {
	existing, err := u.cases.GetLatestByApplicantID(ctx, applicantID)
	if err != nil && !errors.Is(err, domain.ErrVerificationCaseNotFound) {
		return nil, fmt.Errorf("get latest case: %w", err)
	}
	if err == nil && isPending(existing.Status) {
		return nil, ErrCaseAlreadyPending
	}

	docs, err := u.documents.ListByApplicantID(ctx, applicantID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	if !hasAllRequiredTypes(docs) {
		return nil, ErrMissingDocuments
	}

	c := &domain.VerificationCase{
		ID:          uuid.New(),
		ApplicantID: applicantID,
		Status:      domain.StatusSubmitted,
	}
	if err := u.cases.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create case: %w", err)
	}

	if err := u.queue.Enqueue(ctx, c.ID); err != nil {
		return nil, fmt.Errorf("enqueue case: %w", err)
	}

	return c, nil
}

func (u *VerificationUsecase) GetLatestByApplicantID(ctx context.Context, applicantID uuid.UUID) (*domain.VerificationCase, error) {
	return u.cases.GetLatestByApplicantID(ctx, applicantID)
}

func (u *VerificationUsecase) Process(ctx context.Context, caseID uuid.UUID) {
	c, err := u.cases.GetByID(ctx, caseID)
	if err != nil {
		log.Printf("process case %s: get case: %v", caseID, err)
		return
	}

	if err := u.cases.UpdateStatus(ctx, c.ID, domain.StatusInReview); err != nil {
		log.Printf("process case %s: update to in_review: %v", caseID, err)
		return
	}

	docs, err := u.documents.ListByApplicantID(ctx, c.ApplicantID)
	if err != nil {
		log.Printf("process case %s: list documents: %v", caseID, err)
		return
	}

	result, err := u.provider.Verify(ctx, domain.VerificationRequest{
		CaseID:      c.ID,
		ApplicantID: c.ApplicantID,
		Documents:   docs,
	})
	if err != nil {
		log.Printf("process case %s: provider error: %v", caseID, err)
		return
	}

	status := domain.StatusRejected
	if result.Decision == domain.DecisionApproved {
		status = domain.StatusApproved
	}
	if err := u.cases.UpdateStatus(ctx, c.ID, status); err != nil {
		log.Printf("process case %s: update to %s: %v", caseID, status, err)
	}
}

func isPending(s domain.CaseStatus) bool {
	return s == domain.StatusSubmitted || s == domain.StatusInReview
}

func hasAllRequiredTypes(docs []domain.Document) bool {
	present := make(map[domain.DocumentType]bool, len(docs))
	for _, d := range docs {
		present[d.Type] = true
	}
	for _, t := range requiredDocumentTypes {
		if !present[t] {
			return false
		}
	}
	return true
}
