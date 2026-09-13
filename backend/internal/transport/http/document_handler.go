package http

import (
	"errors"
	"log"
	"net/http"
	"time"

	"kyc-verify/internal/domain"
	"kyc-verify/internal/usecase"
)

const maxUploadSize = 20 << 20

type documentHandler struct {
	applicants *usecase.ApplicantUsecase
	documents  *usecase.DocumentUsecase
}

type documentResponse struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	UploadedAt string `json:"uploaded_at"`
}

func toDocumentResponse(d *domain.Document) documentResponse {
	return documentResponse{
		ID:         d.ID.String(),
		Type:       string(d.Type),
		UploadedAt: d.UploadedAt.Format(time.RFC3339),
	}
}

func (h *documentHandler) upload(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireClaims(w, r)
	if !ok {
		return
	}

	applicant, err := h.applicants.GetByUserID(r.Context(), claims.UserID)
	if err != nil {
		writeApplicantError(w, err)
		return
	}

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file")
		return
	}
	defer file.Close()

	docType := domain.DocumentType(r.FormValue("type"))

	doc, err := h.documents.Upload(r.Context(), applicant.ID, docType, header.Filename, file)
	if err != nil {
		writeDocumentError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toDocumentResponse(doc))
}

func (h *documentHandler) listMine(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireClaims(w, r)
	if !ok {
		return
	}

	applicant, err := h.applicants.GetByUserID(r.Context(), claims.UserID)
	if err != nil {
		writeApplicantError(w, err)
		return
	}

	docs, err := h.documents.ListByApplicantID(r.Context(), applicant.ID)
	if err != nil {
		log.Printf("list documents error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]documentResponse, 0, len(docs))
	for i := range docs {
		resp = append(resp, toDocumentResponse(&docs[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeDocumentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidDocumentType):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("document error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
