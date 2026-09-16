package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kyc-verify/internal/domain"
	"kyc-verify/internal/usecase"
)

type verificationHandler struct {
	applicants   *usecase.ApplicantUsecase
	verification *usecase.VerificationUsecase
}

type verificationCaseResponse struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	ApplicantRisk   string `json:"applicant_risk_level,omitempty"`
	PriorRejections *int   `json:"applicant_prior_rejections,omitempty"`
}

type decisionRequest struct {
	Comment string `json:"comment"`
}

type queueItemResponse struct {
	CaseID          string `json:"case_id"`
	ApplicantID     string `json:"applicant_id"`
	FullName        string `json:"full_name"`
	RiskLevel       string `json:"risk_level"`
	PriorRejections int    `json:"prior_rejections"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func toVerificationCaseResponse(c *domain.VerificationCase) verificationCaseResponse {
	return verificationCaseResponse{
		ID:        c.ID.String(),
		Status:    string(c.Status),
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
}

func toQueueItemResponse(item *domain.QueueItem) queueItemResponse {
	return queueItemResponse{
		CaseID:          item.CaseID.String(),
		ApplicantID:     item.ApplicantID.String(),
		FullName:        item.ApplicantFullName,
		RiskLevel:       string(item.ApplicantRisk),
		PriorRejections: item.PriorRejections,
		Status:          string(item.Status),
		CreatedAt:       item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       item.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *verificationHandler) submit(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireClaims(w, r)
	if !ok {
		return
	}

	applicant, err := h.applicants.GetByUserID(r.Context(), claims.UserID)
	if err != nil {
		writeApplicantError(w, err)
		return
	}

	c, err := h.verification.Submit(r.Context(), applicant.ID, claims.UserID)
	if err != nil {
		writeVerificationError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, toVerificationCaseResponse(c))
}

func (h *verificationHandler) getMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireClaims(w, r)
	if !ok {
		return
	}

	applicant, err := h.applicants.GetByUserID(r.Context(), claims.UserID)
	if err != nil {
		writeApplicantError(w, err)
		return
	}

	c, err := h.verification.GetLatestByApplicantID(r.Context(), applicant.ID)
	if err != nil {
		writeVerificationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toVerificationCaseResponse(c))
}

func (h *verificationHandler) listQueue(w http.ResponseWriter, r *http.Request) {
	items, err := h.verification.ListQueue(r.Context())
	if err != nil {
		log.Printf("list queue error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]queueItemResponse, 0, len(items))
	for i := range items {
		resp = append(resp, toQueueItemResponse(&items[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *verificationHandler) getByID(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid case id")
		return
	}

	c, err := h.verification.GetByID(r.Context(), caseID)
	if err != nil {
		writeVerificationError(w, err)
		return
	}

	applicant, err := h.applicants.GetByID(r.Context(), c.ApplicantID)
	if err != nil {
		writeApplicantError(w, err)
		return
	}

	priorRejections, err := h.verification.PriorRejections(r.Context(), c.ApplicantID)
	if err != nil {
		log.Printf("get prior rejections error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := toVerificationCaseResponse(c)
	resp.ApplicantRisk = string(applicant.RiskLevel)
	resp.PriorRejections = &priorRejections
	writeJSON(w, http.StatusOK, resp)
}

func (h *verificationHandler) listDocuments(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid case id")
		return
	}

	docs, err := h.verification.GetDocuments(r.Context(), caseID)
	if err != nil {
		writeVerificationError(w, err)
		return
	}

	resp := make([]documentResponse, 0, len(docs))
	for i := range docs {
		resp = append(resp, toDocumentResponse(&docs[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *verificationHandler) approve(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, h.verification.Approve)
}

func (h *verificationHandler) reject(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, h.verification.Reject)
}

func (h *verificationHandler) decide(
	w http.ResponseWriter,
	r *http.Request,
	decide func(ctx context.Context, caseID, reviewerID uuid.UUID, comment string) (*domain.VerificationCase, error),
) {
	claims, ok := requireClaims(w, r)
	if !ok {
		return
	}

	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid case id")
		return
	}

	var req decisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := decide(r.Context(), caseID, claims.UserID, req.Comment)
	if err != nil {
		writeVerificationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toVerificationCaseResponse(c))
}

func writeVerificationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrMissingDocuments):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrCaseAlreadyPending):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidTransition):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrVerificationCaseNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		log.Printf("verification error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
