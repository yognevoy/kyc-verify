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

const maxAutoRejectAttempts = 2

var requiredDocumentTypes = [...]domain.DocumentType{
	domain.DocumentPassport,
	domain.DocumentSelfie,
	domain.DocumentProofOfAddress,
}

type CaseQueue interface {
	Enqueue(ctx context.Context, caseID uuid.UUID) error
}

type VerificationUsecase struct {
	cases      domain.VerificationCaseRepository
	documents  domain.DocumentRepository
	applicants domain.ApplicantRepository
	provider   domain.VerificationProvider
	queue      CaseQueue
}

func NewVerificationUsecase(
	cases domain.VerificationCaseRepository,
	documents domain.DocumentRepository,
	applicants domain.ApplicantRepository,
	provider domain.VerificationProvider,
	queue CaseQueue,
) *VerificationUsecase {
	return &VerificationUsecase{cases: cases, documents: documents, applicants: applicants, provider: provider, queue: queue}
}

func (u *VerificationUsecase) Submit(ctx context.Context, applicantID, userID uuid.UUID) (*domain.VerificationCase, error) {
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
		Status:      domain.StatusDraft,
	}
	fromStatus := c.Status
	if err := c.TransitionTo(domain.StatusSubmitted); err != nil {
		return nil, fmt.Errorf("transition to submitted: %w", err)
	}

	event := &domain.VerificationCaseEvent{
		ID:         uuid.New(),
		CaseID:     c.ID,
		FromStatus: fromStatus,
		ToStatus:   c.Status,
		ActorType:  domain.ActorUser,
		ActorID:    &userID,
	}
	if err := u.cases.Create(ctx, c, event); err != nil {
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

func (u *VerificationUsecase) GetByID(ctx context.Context, caseID uuid.UUID) (*domain.VerificationCase, error) {
	return u.cases.GetByID(ctx, caseID)
}

func (u *VerificationUsecase) ListQueue(ctx context.Context) ([]domain.QueueItem, error) {
	return u.cases.ListQueue(ctx)
}

func (u *VerificationUsecase) GetDocuments(ctx context.Context, caseID uuid.UUID) ([]domain.Document, error) {
	c, err := u.cases.GetByID(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("get case: %w", err)
	}
	return u.documents.ListByApplicantID(ctx, c.ApplicantID)
}

func (u *VerificationUsecase) PriorRejections(ctx context.Context, applicantID uuid.UUID) (int, error) {
	return u.cases.CountByApplicantIDAndStatus(ctx, applicantID, domain.StatusRejected)
}

func (u *VerificationUsecase) Process(ctx context.Context, caseID uuid.UUID) {
	c, err := u.cases.GetByID(ctx, caseID)
	if err != nil {
		log.Printf("process case %s: get case: %v", caseID, err)
		return
	}

	if err := u.transition(ctx, c, domain.StatusInReview, domain.ActorSystem, nil, nil); err != nil {
		log.Printf("process case %s: transition to in_review: %v", caseID, err)
		return
	}

	applicant, err := u.applicants.GetByID(ctx, c.ApplicantID)
	if err != nil {
		log.Printf("process case %s: get applicant: %v", caseID, err)
		return
	}
	if applicant.RiskLevel != domain.RiskLow {
		return
	}

	priorRejections, err := u.PriorRejections(ctx, c.ApplicantID)
	if err != nil {
		log.Printf("process case %s: count prior rejections: %v", caseID, err)
		return
	}
	if priorRejections >= maxAutoRejectAttempts {
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

	var comment *string
	if result.Reason != "" {
		comment = &result.Reason
	}
	if err := u.transition(ctx, c, status, domain.ActorProvider, nil, comment); err != nil {
		log.Printf("process case %s: transition to %s: %v", caseID, status, err)
	}
}

func (u *VerificationUsecase) Approve(ctx context.Context, caseID, reviewerID uuid.UUID, comment string) (*domain.VerificationCase, error) {
	return u.decide(ctx, caseID, reviewerID, domain.StatusApproved, comment)
}

func (u *VerificationUsecase) Reject(ctx context.Context, caseID, reviewerID uuid.UUID, comment string) (*domain.VerificationCase, error) {
	return u.decide(ctx, caseID, reviewerID, domain.StatusRejected, comment)
}

func (u *VerificationUsecase) decide(ctx context.Context, caseID, reviewerID uuid.UUID, status domain.CaseStatus, comment string) (*domain.VerificationCase, error) {
	c, err := u.cases.GetByID(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("get case: %w", err)
	}

	var commentPtr *string
	if comment != "" {
		commentPtr = &comment
	}

	if err := u.transition(ctx, c, status, domain.ActorReviewer, &reviewerID, commentPtr); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *VerificationUsecase) transition(
	ctx context.Context,
	c *domain.VerificationCase,
	status domain.CaseStatus,
	actorType domain.ActorType,
	actorID *uuid.UUID,
	comment *string,
) error {
	fromStatus := c.Status
	if err := c.TransitionTo(status); err != nil {
		return err
	}

	event := &domain.VerificationCaseEvent{
		ID:         uuid.New(),
		CaseID:     c.ID,
		FromStatus: fromStatus,
		ToStatus:   c.Status,
		ActorType:  actorType,
		ActorID:    actorID,
		Comment:    comment,
	}
	if err := u.cases.Transition(ctx, c, event); err != nil {
		return fmt.Errorf("persist transition: %w", err)
	}
	return nil
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
