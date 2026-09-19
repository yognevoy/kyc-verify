package usecase

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"kyc-verify/internal/domain"
)

type fakeCaseRepo struct {
	mu     sync.Mutex
	cases  map[uuid.UUID]*domain.VerificationCase
	order  []uuid.UUID
	events []domain.VerificationCaseEvent
}

func newFakeCaseRepo() *fakeCaseRepo {
	return &fakeCaseRepo{cases: make(map[uuid.UUID]*domain.VerificationCase)}
}

func (r *fakeCaseRepo) put(c domain.VerificationCase) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cases[c.ID] = &c
	r.order = append(r.order, c.ID)
}

func (r *fakeCaseRepo) get(id uuid.UUID) domain.VerificationCase {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.cases[id]; ok {
		return *c
	}
	return domain.VerificationCase{}
}

func (r *fakeCaseRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.cases)
}

func (r *fakeCaseRepo) eventsFor(id uuid.UUID) []domain.VerificationCaseEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.VerificationCaseEvent
	for _, e := range r.events {
		if e.CaseID == id {
			out = append(out, e)
		}
	}
	return out
}

func (r *fakeCaseRepo) Create(_ context.Context, c *domain.VerificationCase, event *domain.VerificationCaseEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.cases {
		if existing.ApplicantID == c.ApplicantID && !existing.Status.IsTerminal() {
			return domain.ErrActiveCaseExists
		}
	}
	cp := *c
	r.cases[c.ID] = &cp
	r.order = append(r.order, c.ID)
	r.events = append(r.events, *event)
	return nil
}

func (r *fakeCaseRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.VerificationCase, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.cases[id]
	if !ok {
		return nil, domain.ErrVerificationCaseNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *fakeCaseRepo) GetLatestByApplicantID(_ context.Context, applicantID uuid.UUID) (*domain.VerificationCase, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.order) - 1; i >= 0; i-- {
		if c := r.cases[r.order[i]]; c.ApplicantID == applicantID {
			cp := *c
			return &cp, nil
		}
	}
	return nil, domain.ErrVerificationCaseNotFound
}

func (r *fakeCaseRepo) Transition(_ context.Context, c *domain.VerificationCase, event *domain.VerificationCaseEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.cases[c.ID]
	if !ok || stored.Status != event.FromStatus {
		return domain.ErrInvalidTransition
	}
	stored.Status = c.Status
	r.events = append(r.events, *event)
	return nil
}

func (r *fakeCaseRepo) ListQueue(context.Context) ([]domain.QueueItem, error) {
	return nil, nil
}

func (r *fakeCaseRepo) CountByApplicantIDAndStatus(_ context.Context, applicantID uuid.UUID, status domain.CaseStatus) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, c := range r.cases {
		if c.ApplicantID == applicantID && c.Status == status {
			n++
		}
	}
	return n, nil
}

func (r *fakeCaseRepo) SetProviderReference(_ context.Context, caseID uuid.UUID, reference string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.cases[caseID]
	if !ok {
		return domain.ErrVerificationCaseNotFound
	}
	c.ProviderReference = &reference
	return nil
}

func (r *fakeCaseRepo) GetByProviderReference(_ context.Context, reference string) (*domain.VerificationCase, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.cases {
		if c.ProviderReference != nil && *c.ProviderReference == reference {
			cp := *c
			return &cp, nil
		}
	}
	return nil, domain.ErrVerificationCaseNotFound
}

type fakeDocumentRepo struct {
	mu   sync.Mutex
	docs map[uuid.UUID][]domain.Document
}

func newFakeDocumentRepo() *fakeDocumentRepo {
	return &fakeDocumentRepo{docs: make(map[uuid.UUID][]domain.Document)}
}

func (r *fakeDocumentRepo) Create(_ context.Context, doc *domain.Document) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs[doc.ApplicantID] = append(r.docs[doc.ApplicantID], *doc)
	return nil
}

func (r *fakeDocumentRepo) ListByApplicantID(_ context.Context, applicantID uuid.UUID) ([]domain.Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.Document(nil), r.docs[applicantID]...), nil
}

type fakeApplicantRepo struct {
	mu         sync.Mutex
	applicants map[uuid.UUID]domain.Applicant
}

func newFakeApplicantRepo() *fakeApplicantRepo {
	return &fakeApplicantRepo{applicants: make(map[uuid.UUID]domain.Applicant)}
}

func (r *fakeApplicantRepo) put(a domain.Applicant) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.applicants[a.ID] = a
}

func (r *fakeApplicantRepo) Create(_ context.Context, a *domain.Applicant) error {
	r.put(*a)
	return nil
}

func (r *fakeApplicantRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Applicant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.applicants[id]
	if !ok {
		return nil, domain.ErrApplicantNotFound
	}
	return &a, nil
}

func (r *fakeApplicantRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*domain.Applicant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.applicants {
		if a.UserID == userID {
			cp := a
			return &cp, nil
		}
	}
	return nil, domain.ErrApplicantNotFound
}

func (r *fakeApplicantRepo) Update(_ context.Context, a *domain.Applicant) error {
	r.put(*a)
	return nil
}

type fakeQueue struct {
	mu  sync.Mutex
	ids []uuid.UUID
}

func (q *fakeQueue) Enqueue(_ context.Context, caseID uuid.UUID) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.ids = append(q.ids, caseID)
	return nil
}

func (q *fakeQueue) enqueued() []uuid.UUID {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]uuid.UUID(nil), q.ids...)
}

type fakeNotifier struct {
	mu        sync.Mutex
	published []domain.VerificationCase
}

func (n *fakeNotifier) Publish(_ uuid.UUID, c domain.VerificationCase) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.published = append(n.published, c)
}

func (n *fakeNotifier) Subscribe(uuid.UUID) (<-chan domain.VerificationCase, func()) {
	return nil, func() {}
}

func (n *fakeNotifier) publishedCases() []domain.VerificationCase {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]domain.VerificationCase(nil), n.published...)
}

type fakeProvider struct {
	mu       sync.Mutex
	requests []domain.VerificationRequest
	err      error
}

func (p *fakeProvider) Submit(_ context.Context, req domain.VerificationRequest) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests = append(p.requests, req)
	if p.err != nil {
		return "", p.err
	}
	return fmt.Sprintf("ref-%d", len(p.requests)), nil
}

func (p *fakeProvider) submitted() []domain.VerificationRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]domain.VerificationRequest(nil), p.requests...)
}
