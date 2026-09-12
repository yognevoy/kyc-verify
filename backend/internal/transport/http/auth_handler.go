package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"kyc-verify/internal/domain"
	"kyc-verify/internal/usecase"
)

const refreshCookieName = "refresh_token"

type authHandler struct {
	auth         *usecase.AuthUsecase
	refreshTTL   time.Duration
	cookieSecure bool
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.auth.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	h.setRefreshCookie(w, pair.RefreshToken)
	writeJSON(w, http.StatusCreated, authResponse{AccessToken: pair.AccessToken})
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	h.setRefreshCookie(w, pair.RefreshToken)
	writeJSON(w, http.StatusOK, authResponse{AccessToken: pair.AccessToken})
}

func (h *authHandler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	pair, err := h.auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidRefreshToken) {
			h.clearRefreshCookie(w)
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		log.Printf("refresh error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.setRefreshCookie(w, pair.RefreshToken)
	writeJSON(w, http.StatusOK, authResponse{AccessToken: pair.AccessToken})
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil && cookie.Value != "" {
		if err := h.auth.Logout(r.Context(), cookie.Value); err != nil {
			log.Printf("logout error: %v", err)
		}
	}

	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.refreshTTL.Seconds()),
	})
}

func (h *authHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
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
	claims, ok := requireClaims(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id": claims.UserID.String(),
		"role":    claims.Role,
	})
}
