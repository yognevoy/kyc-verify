package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"kyc-verify/internal/domain"
)

type fakeCallbackUsecase struct {
	err          error
	gotReference string
	gotResult    domain.VerificationResult
	calls        int
}

func (f *fakeCallbackUsecase) HandleProviderCallback(_ context.Context, reference string, result domain.VerificationResult) error {
	f.calls++
	f.gotReference = reference
	f.gotResult = result
	return f.err
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestProviderCallbackHandler(t *testing.T) {
	const secret = "test-secret"

	tests := []struct {
		name       string
		body       string
		signWith   string
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "valid signature and payload is accepted",
			body:       `{"reference":"ref-1","decision":"approved"}`,
			signWith:   secret,
			wantStatus: http.StatusNoContent,
			wantCalls:  1,
		},
		{
			name:       "wrong secret is rejected before reaching usecase",
			body:       `{"reference":"ref-1","decision":"approved"}`,
			signWith:   "wrong-secret",
			wantStatus: http.StatusUnauthorized,
			wantCalls:  0,
		},
		{
			name:       "missing decision is rejected",
			body:       `{"reference":"ref-1","decision":"maybe"}`,
			signWith:   secret,
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
		},
		{
			name:       "missing reference is rejected",
			body:       `{"decision":"approved"}`,
			signWith:   secret,
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usecase := &fakeCallbackUsecase{}
			h := &providerCallbackHandler{usecase: usecase, secret: secret}

			req := httptest.NewRequest(http.MethodPost, "/api/provider-callback", bytes.NewBufferString(tt.body))
			req.Header.Set("X-Signature", sign([]byte(tt.body), tt.signWith))
			rec := httptest.NewRecorder()

			h.handle(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if usecase.calls != tt.wantCalls {
				t.Fatalf("usecase called %d times, want %d", usecase.calls, tt.wantCalls)
			}
		})
	}
}

func TestProviderCallbackHandler_RepeatedCallbackIsIdempotentAtUsecaseLevel(t *testing.T) {
	usecase := &fakeCallbackUsecase{}
	h := &providerCallbackHandler{usecase: usecase, secret: "s"}

	body := `{"reference":"ref-1","decision":"rejected","reason":"blurry document"}`

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/provider-callback", bytes.NewBufferString(body))
		req.Header.Set("X-Signature", sign([]byte(body), "s"))
		rec := httptest.NewRecorder()

		h.handle(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("call %d: status = %d, want %d", i, rec.Code, http.StatusNoContent)
		}
	}

	if usecase.calls != 2 {
		t.Fatalf("usecase called %d times, want 2 (idempotency itself is enforced by VerificationUsecase.HandleProviderCallback, not the handler)", usecase.calls)
	}
	if usecase.gotReference != "ref-1" || usecase.gotResult.Decision != domain.DecisionRejected || usecase.gotResult.Reason != "blurry document" {
		t.Fatalf("unexpected forwarded call: reference=%q result=%+v", usecase.gotReference, usecase.gotResult)
	}
}
