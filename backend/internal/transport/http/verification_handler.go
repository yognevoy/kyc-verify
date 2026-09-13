package http

import (
	"errors"
	"log"
	"net/http"
	"time"

	"kyc-verify/internal/domain"
	"kyc-verify/internal/usecase"
)

type verificationHandler struct {
	applicants   *usecase.ApplicantUsecase
	verification *usecase.VerificationUsecase
}

type verificationCaseResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toVerificationCaseResponse(c *domain.VerificationCase) verificationCaseResponse {
	return verificationCaseResponse{
		ID:        c.ID.String(),
		Status:    string(c.Status),
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
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

	c, err := h.verification.Submit(r.Context(), applicant.ID)
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

func writeVerificationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrMissingDocuments):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrCaseAlreadyPending):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrVerificationCaseNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		log.Printf("verification error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
