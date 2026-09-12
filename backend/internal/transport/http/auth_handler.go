package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"kyc-verify/internal/domain"
	"kyc-verify/internal/usecase"
)

type authHandler struct {
	auth *usecase.AuthUsecase
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.auth.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{Token: token})
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token})
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidEmail), errors.Is(err, usecase.ErrPasswordTooShort):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrUserAlreadyExists):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("auth error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing claims")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id": claims.UserID.String(),
		"role":    claims.Role,
	})
}
