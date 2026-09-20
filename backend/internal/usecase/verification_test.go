package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"kyc-verify/internal/domain"
	"kyc-verify/internal/worker"
)

var errProviderDown = errors.New("provider down")

var allDocumentTypes = []domain.DocumentType{
	domain.DocumentPassport,
	domain.DocumentSelfie,
	domain.DocumentProofOfAddress,
}

type fixture struct {
	uc         *VerificationUsecase
	cases      *fakeCaseRepo
	documents  *fakeDocumentRepo
	applicants *fakeApplicantRepo
	provider   *fakeProvider
	queue      *fakeQueue
	notifier   *fakeNotifier
}

func newFixture() *fixture {
	f := &fixture{
		cases:      newFakeCaseRepo(),
		documents:  newFakeDocumentRepo(),
		applicants: newFakeApplicantRepo(),
		provider:   &fakeProvider{},
		queue:      &fakeQueue{},
		notifier:   &fakeNotifier{},
	}
	f.uc = NewVerificationUsecase(f.cases, f.documents, f.applicants, f.provider, f.queue, f.notifier)
	return f
}

func (f *fixture) applicant(risk domain.RiskLevel) uuid.UUID {
	id := uuid.New()
	f.applicants.put(domain.Applicant{ID: id, RiskLevel: risk})
	return id
}

func (f *fixture) withDocuments(applicantID uuid.UUID, types ...domain.DocumentType) {
	for _, t := range types {
		_ = f.documents.Create(context.Background(), &domain.Document{ID: uuid.New(), ApplicantID: applicantID, Type: t})
	}
}

func (f *fixture) caseWith(applicantID uuid.UUID, status domain.CaseStatus) uuid.UUID {
	id := uuid.New()
	f.cases.put(domain.VerificationCase{ID: id, ApplicantID: applicantID, Status: status})
	return id
}

func (f *fixture) caseWithReference(applicantID uuid.UUID, status domain.CaseStatus, reference string) uuid.UUID {
	id := uuid.New()
	f.cases.put(domain.VerificationCase{ID: id, ApplicantID: applicantID, Status: status, ProviderReference: &reference})
	return id
}

func TestVerificationUsecase_Submit(t *testing.T) {
	tests := []struct {
		name     string
		docs     []domain.DocumentType
		existing domain.CaseStatus
		wantErr  error
	}{
		{name: "all documents and no previous case", docs: allDocumentTypes},
		{name: "no documents", docs: nil, wantErr: ErrMissingDocuments},
		{name: "selfie is missing", docs: []domain.DocumentType{domain.DocumentPassport, domain.DocumentProofOfAddress}, wantErr: ErrMissingDocuments},
		{name: "previous case is submitted", docs: allDocumentTypes, existing: domain.StatusSubmitted, wantErr: ErrCaseAlreadyPending},
		{name: "previous case is in review", docs: allDocumentTypes, existing: domain.StatusInReview, wantErr: ErrCaseAlreadyPending},
		{name: "previous case was rejected", docs: allDocumentTypes, existing: domain.StatusRejected},
		{name: "previous case was approved", docs: allDocumentTypes, existing: domain.StatusApproved},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			applicantID := f.applicant(domain.RiskLow)
			f.withDocuments(applicantID, tt.docs...)
			if tt.existing != "" {
				f.caseWith(applicantID, tt.existing)
			}
			casesBefore := f.cases.count()
			userID := uuid.New()

			c, err := f.uc.Submit(context.Background(), applicantID, userID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Submit() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if c != nil {
					t.Fatalf("Submit() returned a case on error: %+v", c)
				}
				if f.cases.count() != casesBefore {
					t.Fatal("a case was created despite the error")
				}
				if len(f.queue.enqueued()) != 0 || len(f.notifier.publishedCases()) != 0 {
					t.Fatal("rejected submit must not enqueue or notify")
				}
				return
			}

			if c.Status != domain.StatusSubmitted {
				t.Fatalf("status = %s, want %s", c.Status, domain.StatusSubmitted)
			}
			if f.cases.count() != casesBefore+1 {
				t.Fatalf("cases = %d, want %d", f.cases.count(), casesBefore+1)
			}
			events := f.cases.eventsFor(c.ID)
			if len(events) != 1 {
				t.Fatalf("events = %d, want 1", len(events))
			}
			e := events[0]
			if e.FromStatus != domain.StatusDraft || e.ToStatus != domain.StatusSubmitted || e.ActorType != domain.ActorUser || e.ActorID == nil || *e.ActorID != userID {
				t.Fatalf("unexpected audit event: %+v", e)
			}
			if got := f.queue.enqueued(); len(got) != 1 || got[0] != c.ID {
				t.Fatalf("enqueued = %v, want [%s]", got, c.ID)
			}
			if got := f.notifier.publishedCases(); len(got) != 1 || got[0].Status != domain.StatusSubmitted {
				t.Fatalf("published = %+v, want one submitted case", got)
			}
		})
	}
}

type staleLatestRepo struct {
	*fakeCaseRepo
}

func (staleLatestRepo) GetLatestByApplicantID(context.Context, uuid.UUID) (*domain.VerificationCase, error) {
	return nil, domain.ErrVerificationCaseNotFound
}

func TestVerificationUsecase_Submit_ActiveCaseConstraintIsPending(t *testing.T) {
	f := newFixture()
	applicantID := f.applicant(domain.RiskLow)
	f.withDocuments(applicantID, allDocumentTypes...)
	f.caseWith(applicantID, domain.StatusInReview)
	uc := NewVerificationUsecase(staleLatestRepo{f.cases}, f.documents, f.applicants, f.provider, f.queue, f.notifier)

	c, err := uc.Submit(context.Background(), applicantID, uuid.New())

	if !errors.Is(err, ErrCaseAlreadyPending) {
		t.Fatalf("Submit() error = %v, want %v", err, ErrCaseAlreadyPending)
	}
	if c != nil || f.cases.count() != 1 {
		t.Fatalf("a second active case must not be created: case=%+v, cases=%d", c, f.cases.count())
	}
	if len(f.queue.enqueued()) != 0 || len(f.notifier.publishedCases()) != 0 {
		t.Fatal("rejected submit must not enqueue or notify")
	}
}

func TestVerificationUsecase_Submit_ConcurrentRequestsCreateOneCase(t *testing.T) {
	const requests = 20

	f := newFixture()
	applicantID := f.applicant(domain.RiskLow)
	f.withDocuments(applicantID, allDocumentTypes...)

	start := make(chan struct{})
	errs := make([]error, requests)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = f.uc.Submit(context.Background(), applicantID, uuid.New())
		}()
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for i, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case !errors.Is(err, ErrCaseAlreadyPending):
			t.Fatalf("request %d: error = %v, want nil or %v", i, err, ErrCaseAlreadyPending)
		}
	}
	if succeeded != 1 {
		t.Fatalf("succeeded = %d, want exactly 1", succeeded)
	}
	if f.cases.count() != 1 || len(f.queue.enqueued()) != 1 || len(f.notifier.publishedCases()) != 1 {
		t.Fatalf("cases=%d enqueued=%d published=%d, want 1 each", f.cases.count(), len(f.queue.enqueued()), len(f.notifier.publishedCases()))
	}
}

func TestVerificationUsecase_Process(t *testing.T) {
	tests := []struct {
		name             string
		risk             domain.RiskLevel
		priorRejections  int
		providerErr      error
		wantProviderCall bool
		wantReference    bool
	}{
		{name: "low risk without history goes to provider", risk: domain.RiskLow, wantProviderCall: true, wantReference: true},
		{name: "medium risk waits for manual review", risk: domain.RiskMedium},
		{name: "high risk waits for manual review", risk: domain.RiskHigh},
		{name: "one prior rejection is still automatic", risk: domain.RiskLow, priorRejections: maxAutoRejectAttempts - 1, wantProviderCall: true, wantReference: true},
		{name: "prior rejections at the limit require manual review", risk: domain.RiskLow, priorRejections: maxAutoRejectAttempts},
		{name: "prior rejections above the limit require manual review", risk: domain.RiskLow, priorRejections: maxAutoRejectAttempts + 1},
		{name: "provider failure leaves case in review without reference", risk: domain.RiskLow, providerErr: errProviderDown, wantProviderCall: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			f.provider.err = tt.providerErr
			applicantID := f.applicant(tt.risk)
			f.withDocuments(applicantID, allDocumentTypes...)
			for i := 0; i < tt.priorRejections; i++ {
				f.caseWith(applicantID, domain.StatusRejected)
			}
			caseID := f.caseWith(applicantID, domain.StatusSubmitted)

			f.uc.Process(context.Background(), caseID)

			stored := f.cases.get(caseID)
			if stored.Status != domain.StatusInReview {
				t.Fatalf("status = %s, want %s", stored.Status, domain.StatusInReview)
			}
			events := f.cases.eventsFor(caseID)
			if len(events) != 1 || events[0].FromStatus != domain.StatusSubmitted || events[0].ToStatus != domain.StatusInReview || events[0].ActorType != domain.ActorSystem {
				t.Fatalf("unexpected audit events: %+v", events)
			}

			requests := f.provider.submitted()
			if (len(requests) == 1) != tt.wantProviderCall || len(requests) > 1 {
				t.Fatalf("provider calls = %d, want called=%v", len(requests), tt.wantProviderCall)
			}
			if tt.wantProviderCall {
				r := requests[0]
				if r.CaseID != caseID || r.ApplicantID != applicantID || len(r.Documents) != len(allDocumentTypes) {
					t.Fatalf("unexpected provider request: %+v", r)
				}
			}
			if hasReference := stored.ProviderReference != nil; hasReference != tt.wantReference {
				t.Fatalf("provider reference set = %v, want %v", hasReference, tt.wantReference)
			}
		})
	}
}

func TestVerificationUsecase_Process_UnknownCaseIsIgnored(t *testing.T) {
	f := newFixture()

	f.uc.Process(context.Background(), uuid.New())

	if len(f.provider.submitted()) != 0 {
		t.Fatal("provider must not be called for an unknown case")
	}
}

func TestVerificationUsecase_Process_DuplicateJobIsIgnored(t *testing.T) {
	f := newFixture()
	applicantID := f.applicant(domain.RiskLow)
	f.withDocuments(applicantID, allDocumentTypes...)
	caseID := f.caseWith(applicantID, domain.StatusInReview)

	f.uc.Process(context.Background(), caseID)

	if len(f.provider.submitted()) != 0 {
		t.Fatal("provider must not be called again for a case already in review")
	}
	if got := len(f.cases.eventsFor(caseID)); got != 0 {
		t.Fatalf("events = %d, want 0", got)
	}
}

func TestVerificationUsecase_HandleProviderCallback(t *testing.T) {
	tests := []struct {
		name        string
		status      domain.CaseStatus
		reference   string
		result      domain.VerificationResult
		wantErr     error
		wantStatus  domain.CaseStatus
		wantEvents  int
		wantComment string
	}{
		{
			name: "approval moves case to approved", status: domain.StatusInReview, reference: "ref-1",
			result:     domain.VerificationResult{Decision: domain.DecisionApproved},
			wantStatus: domain.StatusApproved, wantEvents: 1,
		},
		{
			name: "rejection keeps the provider reason", status: domain.StatusInReview, reference: "ref-1",
			result:     domain.VerificationResult{Decision: domain.DecisionRejected, Reason: "blurry document"},
			wantStatus: domain.StatusRejected, wantEvents: 1, wantComment: "blurry document",
		},
		{
			name: "repeated approval is a no-op", status: domain.StatusApproved, reference: "ref-1",
			result:     domain.VerificationResult{Decision: domain.DecisionApproved},
			wantStatus: domain.StatusApproved,
		},
		{
			name: "conflicting late decision does not override a final one", status: domain.StatusRejected, reference: "ref-1",
			result:     domain.VerificationResult{Decision: domain.DecisionApproved},
			wantStatus: domain.StatusRejected,
		},
		{
			name: "unknown reference is an error", status: domain.StatusInReview, reference: "other-ref",
			result:  domain.VerificationResult{Decision: domain.DecisionApproved},
			wantErr: domain.ErrVerificationCaseNotFound, wantStatus: domain.StatusInReview,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			caseID := f.caseWithReference(f.applicant(domain.RiskLow), tt.status, "ref-1")

			err := f.uc.HandleProviderCallback(context.Background(), tt.reference, tt.result)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("HandleProviderCallback() error = %v, want %v", err, tt.wantErr)
			}
			if got := f.cases.get(caseID).Status; got != tt.wantStatus {
				t.Fatalf("status = %s, want %s", got, tt.wantStatus)
			}
			events := f.cases.eventsFor(caseID)
			if len(events) != tt.wantEvents {
				t.Fatalf("events = %d, want %d", len(events), tt.wantEvents)
			}
			if got := len(f.notifier.publishedCases()); got != tt.wantEvents {
				t.Fatalf("published = %d, want %d", got, tt.wantEvents)
			}
			if tt.wantEvents == 0 {
				return
			}
			e := events[0]
			if e.ActorType != domain.ActorProvider || e.ActorID != nil || e.FromStatus != domain.StatusInReview {
				t.Fatalf("unexpected audit event: %+v", e)
			}
			if tt.wantComment == "" && e.Comment != nil {
				t.Fatalf("comment = %q, want none", *e.Comment)
			}
			if tt.wantComment != "" && (e.Comment == nil || *e.Comment != tt.wantComment) {
				t.Fatalf("comment = %v, want %q", e.Comment, tt.wantComment)
			}
		})
	}
}

func TestVerificationUsecase_ApproveReject(t *testing.T) {
	tests := []struct {
		name        string
		approve     bool
		status      domain.CaseStatus
		unknownCase bool
		ownCase     bool
		comment     string
		wantErr     error
		wantStatus  domain.CaseStatus
	}{
		{name: "reviewer cannot approve their own case", approve: true, status: domain.StatusInReview, ownCase: true, wantErr: ErrSelfReview, wantStatus: domain.StatusInReview},
		{name: "reviewer cannot reject their own case", approve: false, status: domain.StatusInReview, ownCase: true, wantErr: ErrSelfReview, wantStatus: domain.StatusInReview},
		{name: "reviewer approves a case in review", approve: true, status: domain.StatusInReview, comment: "documents match", wantStatus: domain.StatusApproved},
		{name: "reviewer rejects a case in review without comment", approve: false, status: domain.StatusInReview, wantStatus: domain.StatusRejected},
		{name: "case that was not picked up yet cannot be decided", approve: true, status: domain.StatusSubmitted, wantErr: domain.ErrInvalidTransition, wantStatus: domain.StatusSubmitted},
		{name: "approved case cannot be rejected", approve: false, status: domain.StatusApproved, wantErr: domain.ErrInvalidTransition, wantStatus: domain.StatusApproved},
		{name: "rejected case cannot be approved", approve: true, status: domain.StatusRejected, wantErr: domain.ErrInvalidTransition, wantStatus: domain.StatusRejected},
		{name: "unknown case", approve: true, unknownCase: true, wantErr: domain.ErrVerificationCaseNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			caseID := uuid.New()
			reviewerID := uuid.New()
			if !tt.unknownCase {
				applicantID := f.applicant(domain.RiskLow)
				if tt.ownCase {
					f.applicants.put(domain.Applicant{ID: applicantID, UserID: reviewerID, RiskLevel: domain.RiskLow})
				}
				caseID = f.caseWith(applicantID, tt.status)
			}

			decide := f.uc.Reject
			if tt.approve {
				decide = f.uc.Approve
			}
			c, err := decide(context.Background(), caseID, reviewerID, tt.comment)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if c != nil {
					t.Fatalf("returned a case on error: %+v", c)
				}
				if !tt.unknownCase {
					if got := f.cases.get(caseID).Status; got != tt.wantStatus {
						t.Fatalf("status = %s, want unchanged %s", got, tt.wantStatus)
					}
					if len(f.cases.eventsFor(caseID)) != 0 {
						t.Fatal("failed decision must not write an audit event")
					}
				}
				return
			}

			if c.Status != tt.wantStatus || f.cases.get(caseID).Status != tt.wantStatus {
				t.Fatalf("status = %s (stored %s), want %s", c.Status, f.cases.get(caseID).Status, tt.wantStatus)
			}
			events := f.cases.eventsFor(caseID)
			if len(events) != 1 {
				t.Fatalf("events = %d, want 1", len(events))
			}
			e := events[0]
			if e.ActorType != domain.ActorReviewer || e.ActorID == nil || *e.ActorID != reviewerID || e.FromStatus != domain.StatusInReview || e.ToStatus != tt.wantStatus {
				t.Fatalf("unexpected audit event: %+v", e)
			}
			if tt.comment == "" && e.Comment != nil {
				t.Fatalf("comment = %q, want none", *e.Comment)
			}
			if tt.comment != "" && (e.Comment == nil || *e.Comment != tt.comment) {
				t.Fatalf("comment = %v, want %q", e.Comment, tt.comment)
			}
		})
	}
}

func TestVerificationUsecase_HandleProviderCallback_ConcurrentDuplicates(t *testing.T) {
	const deliveries = 50

	f := newFixture()
	caseID := f.caseWithReference(f.applicant(domain.RiskLow), domain.StatusInReview, "ref-1")

	start := make(chan struct{})
	errs := make([]error, deliveries)
	var wg sync.WaitGroup
	for i := 0; i < deliveries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = f.uc.HandleProviderCallback(context.Background(), "ref-1", domain.VerificationResult{Decision: domain.DecisionApproved})
		}()
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("delivery %d returned %v, want nil", i, err)
		}
	}
	if got := f.cases.get(caseID).Status; got != domain.StatusApproved {
		t.Fatalf("status = %s, want %s", got, domain.StatusApproved)
	}
	if got := len(f.cases.eventsFor(caseID)); got != 1 {
		t.Fatalf("events = %d, want exactly 1", got)
	}
	if got := len(f.notifier.publishedCases()); got != 1 {
		t.Fatalf("published = %d, want exactly 1", got)
	}
}

func TestVerificationUsecase_ReviewerAndProviderRace(t *testing.T) {
	const rounds = 200

	for round := 0; round < rounds; round++ {
		f := newFixture()
		caseID := f.caseWithReference(f.applicant(domain.RiskLow), domain.StatusInReview, "ref-1")
		reviewerID := uuid.New()

		var approveErr, callbackErr error
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, approveErr = f.uc.Approve(context.Background(), caseID, reviewerID, "")
		}()
		go func() {
			defer wg.Done()
			<-start
			callbackErr = f.uc.HandleProviderCallback(context.Background(), "ref-1", domain.VerificationResult{Decision: domain.DecisionRejected})
		}()
		close(start)
		wg.Wait()

		if callbackErr != nil {
			t.Fatalf("round %d: callback error = %v, want nil", round, callbackErr)
		}
		events := f.cases.eventsFor(caseID)
		if len(events) != 1 {
			t.Fatalf("round %d: events = %d, want exactly 1", round, len(events))
		}
		e := events[0]
		final := f.cases.get(caseID).Status
		if final != e.ToStatus {
			t.Fatalf("round %d: final status %s does not match the only event %+v", round, final, e)
		}

		switch e.ActorType {
		case domain.ActorReviewer:
			if approveErr != nil || final != domain.StatusApproved {
				t.Fatalf("round %d: reviewer won but approveErr=%v final=%s", round, approveErr, final)
			}
		case domain.ActorProvider:
			if !errors.Is(approveErr, domain.ErrInvalidTransition) || final != domain.StatusRejected {
				t.Fatalf("round %d: provider won but approveErr=%v final=%s", round, approveErr, final)
			}
		default:
			t.Fatalf("round %d: unexpected actor in %+v", round, e)
		}
	}
}

func TestVerificationUsecase_Process_ThroughWorkerPool(t *testing.T) {
	const cases = 100
	const workers = 8

	f := newFixture()
	ids := make([]uuid.UUID, cases)
	for i := range ids {
		applicantID := f.applicant(domain.RiskLow)
		f.withDocuments(applicantID, allDocumentTypes...)
		ids[i] = f.caseWith(applicantID, domain.StatusSubmitted)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool := worker.NewPool(2*cases, func(ctx context.Context, id uuid.UUID) {
		f.uc.Process(ctx, id)
	})
	pool.Start(ctx, workers)
	for _, id := range ids {
		for attempt := 0; attempt < 2; attempt++ {
			if err := pool.Enqueue(ctx, id); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
		}
	}
	pool.Stop()

	if got := len(f.provider.submitted()); got != cases {
		t.Fatalf("provider calls = %d, want %d (duplicate jobs must not reach the provider)", got, cases)
	}
	references := make(map[string]bool, cases)
	for _, id := range ids {
		stored := f.cases.get(id)
		if stored.Status != domain.StatusInReview {
			t.Fatalf("case %s status = %s, want %s", id, stored.Status, domain.StatusInReview)
		}
		if stored.ProviderReference == nil {
			t.Fatalf("case %s has no provider reference", id)
		}
		references[*stored.ProviderReference] = true
		if got := len(f.cases.eventsFor(id)); got != 1 {
			t.Fatalf("case %s events = %d, want 1", id, got)
		}
	}
	if len(references) != cases {
		t.Fatalf("distinct references = %d, want %d", len(references), cases)
	}
}

func TestVerificationUsecase_RequeueSubmitted(t *testing.T) {
	f := newFixture()
	applicantID := f.applicant(domain.RiskLow)
	statuses := []domain.CaseStatus{
		domain.StatusSubmitted,
		domain.StatusInReview,
		domain.StatusApproved,
		domain.StatusRejected,
		domain.StatusSubmitted,
	}
	var want []uuid.UUID
	for _, status := range statuses {
		c := domain.VerificationCase{ID: uuid.New(), ApplicantID: applicantID, Status: status}
		f.cases.put(c)
		if status == domain.StatusSubmitted {
			want = append(want, c.ID)
		}
	}

	if err := f.uc.RequeueSubmitted(context.Background()); err != nil {
		t.Fatalf("RequeueSubmitted: %v", err)
	}

	got := f.queue.enqueued()
	if len(got) != len(want) {
		t.Fatalf("enqueued = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("enqueued = %v, want %v", got, want)
		}
	}
}
