package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"kyc-verify/internal/domain"
	"kyc-verify/internal/usecase"
)

type applicantHandler struct {
	applicants *usecase.ApplicantUsecase
}

type applicantRequest struct {
	FullName  string `json:"full_name"`
	BirthDate string `json:"birth_date"`
	Country   string `json:"country"`
}

type applicantResponse struct {
	ID        string `json:"id"`
	FullName  string `json:"full_name"`
	BirthDate string `json:"birth_date"`
	Country   string `json:"country"`
	RiskLevel string `json:"risk_level"`
}

func toApplicantResponse(a *domain.Applicant) applicantResponse {
	return applicantResponse{
		ID:        a.ID.String(),
		FullName:  a.FullName,
		BirthDate: a.BirthDate.Format(usecase.BirthDateLayout),
		Country:   a.Country,
		RiskLevel: string(a.RiskLevel),
	}
}

func (h *applicantHandler) create(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireClaims(w, r)
	if !ok {
		return
	}

	var req applicantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	applicant, err := h.applicants.Create(r.Context(), claims.UserID, usecase.ApplicantInput{
		FullName:  req.FullName,
		BirthDate: req.BirthDate,
		Country:   req.Country,
	})
	if err != nil {
		writeApplicantError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toApplicantResponse(applicant))
}

func (h *applicantHandler) getMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireClaims(w, r)
	if !ok {
		return
	}

	applicant, err := h.applicants.GetByUserID(r.Context(), claims.UserID)
	if err != nil {
		writeApplicantError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toApplicantResponse(applicant))
}

func (h *applicantHandler) updateMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireClaims(w, r)
	if !ok {
		return
	}

	var req applicantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	applicant, err := h.applicants.Update(r.Context(), claims.UserID, usecase.ApplicantInput{
		FullName:  req.FullName,
		BirthDate: req.BirthDate,
		Country:   req.Country,
	})
	if err != nil {
		writeApplicantError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toApplicantResponse(applicant))
}

func writeApplicantError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrFullNameRequired),
		errors.Is(err, usecase.ErrInvalidBirthDate),
		errors.Is(err, usecase.ErrInvalidCountry):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrApplicantAlreadyExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrApplicantNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		log.Printf("applicant error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
