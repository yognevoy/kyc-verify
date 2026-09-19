package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"kyc-verify/internal/domain"
)

func TestHTTPProvider_Submit(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		wantRefRef string
	}{
		{
			name: "returns reference on success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				var req httpSubmitRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if req.CaseID == "" || req.ApplicantID == "" || len(req.Documents) != 1 {
					t.Fatalf("unexpected request body: %+v", req)
				}
				json.NewEncoder(w).Encode(httpSubmitResponse{Reference: "ref-123"})
			},
			wantRefRef: "ref-123",
		},
		{
			name: "non-2xx status is an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantErr: true,
		},
		{
			name: "empty reference in response is an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(httpSubmitResponse{})
			},
			wantErr: true,
		},
		{
			name: "malformed response body is an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("not json"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			p := NewHTTPProvider(srv.URL)
			reference, err := p.Submit(context.Background(), domain.VerificationRequest{
				CaseID:      uuid.New(),
				ApplicantID: uuid.New(),
				Documents:   []domain.Document{{ID: uuid.New(), Type: domain.DocumentPassport}},
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("Submit() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Submit() error = %v, want nil", err)
			}
			if reference != tt.wantRefRef {
				t.Fatalf("Submit() reference = %q, want %q", reference, tt.wantRefRef)
			}
		})
	}
}

func TestHTTPProvider_Submit_UnreachableServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	p := NewHTTPProvider(url)
	if _, err := p.Submit(context.Background(), domain.VerificationRequest{CaseID: uuid.New(), ApplicantID: uuid.New()}); err == nil {
		t.Fatal("Submit() error = nil, want error for unreachable server")
	}
}
