package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"kyc-verify/internal/domain"
)

type providerCallbackUsecase interface {
	HandleProviderCallback(ctx context.Context, reference string, result domain.VerificationResult) error
}

type providerCallbackHandler struct {
	usecase providerCallbackUsecase
	secret  string
}

type providerCallbackRequest struct {
	Reference string `json:"reference"`
	Decision  string `json:"decision"`
	Reason    string `json:"reason"`
}

func (h *providerCallbackHandler) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !validCallbackSignature(body, r.Header.Get("X-Signature"), h.secret) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	var req providerCallbackRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	decision, ok := parseCallbackDecision(req.Decision)
	if req.Reference == "" || !ok {
		writeError(w, http.StatusBadRequest, "invalid callback payload")
		return
	}

	err = h.usecase.HandleProviderCallback(r.Context(), req.Reference, domain.VerificationResult{
		Decision: decision,
		Reason:   req.Reason,
	})
	if err != nil {
		if errors.Is(err, domain.ErrVerificationCaseNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		log.Printf("provider callback error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validCallbackSignature(body []byte, signatureHeader, secret string) bool {
	got, err := hex.DecodeString(signatureHeader)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), got)
}

func parseCallbackDecision(s string) (domain.VerificationDecision, bool) {
	d := domain.VerificationDecision(s)
	if d == domain.DecisionApproved || d == domain.DecisionRejected {
		return d, true
	}
	return "", false
}
