package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"kyc-verify/internal/domain"
)

type fakeStorage struct {
	files map[string]string
}

func (s *fakeStorage) Save(context.Context, uuid.UUID, domain.DocumentType, string, io.Reader) (string, error) {
	return "", errors.New("not implemented")
}

func (s *fakeStorage) Open(_ context.Context, path string) (io.ReadCloser, error) {
	content, ok := s.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func TestHTTPProvider_Submit(t *testing.T) {
	passport := domain.Document{ID: uuid.New(), Type: domain.DocumentPassport, FilePath: "/uploads/passport.jpg"}
	selfie := domain.Document{ID: uuid.New(), Type: domain.DocumentSelfie, FilePath: "/uploads/selfie.png"}
	storage := &fakeStorage{files: map[string]string{
		passport.FilePath: "passport-bytes",
		selfie.FilePath:   "selfie-bytes",
	}}

	tests := []struct {
		name      string
		documents []domain.Document
		handler   http.HandlerFunc
		wantRef   string
		wantErr   bool
	}{
		{
			name:      "sends ids and files as multipart, returns reference",
			documents: []domain.Document{passport, selfie},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Errorf("parse multipart: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				for _, field := range []string{"case_id", "applicant_id"} {
					if v := r.MultipartForm.Value[field]; len(v) != 1 || v[0] == "" {
						t.Errorf("field %q = %v, want one non-empty value", field, v)
					}
				}

				for field, want := range map[string]string{"passport": "passport-bytes", "selfie": "selfie-bytes"} {
					headers := r.MultipartForm.File[field]
					if len(headers) != 1 {
						t.Errorf("field %q: got %d files, want 1", field, len(headers))
						continue
					}
					f, err := headers[0].Open()
					if err != nil {
						t.Errorf("open %q: %v", field, err)
						continue
					}
					got, _ := io.ReadAll(f)
					f.Close()
					if string(got) != want {
						t.Errorf("field %q content = %q, want %q", field, got, want)
					}
				}
				if name := r.MultipartForm.File["passport"][0].Filename; !strings.HasSuffix(name, ".jpg") || !strings.HasPrefix(name, passport.ID.String()) {
					t.Errorf("passport filename = %q, want <id>.jpg", name)
				}

				json.NewEncoder(w).Encode(httpSubmitResponse{Reference: "ref-123"})
			},
			wantRef: "ref-123",
		},
		{
			name:      "non-2xx status is an error",
			documents: []domain.Document{passport},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantErr: true,
		},
		{
			name:      "empty reference in response is an error",
			documents: []domain.Document{passport},
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(httpSubmitResponse{})
			},
			wantErr: true,
		},
		{
			name:      "malformed response body is an error",
			documents: []domain.Document{passport},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("not json"))
			},
			wantErr: true,
		},
		{
			name:      "unreadable document fails before a reference is returned",
			documents: []domain.Document{{ID: uuid.New(), Type: domain.DocumentPassport, FilePath: "/uploads/missing.jpg"}},
			handler: func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				json.NewEncoder(w).Encode(httpSubmitResponse{Reference: "ref-should-not-be-used"})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			p := NewHTTPProvider(srv.URL, storage)
			reference, err := p.Submit(context.Background(), domain.VerificationRequest{
				CaseID:      uuid.New(),
				ApplicantID: uuid.New(),
				Documents:   tt.documents,
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
			if reference != tt.wantRef {
				t.Fatalf("Submit() reference = %q, want %q", reference, tt.wantRef)
			}
		})
	}
}

func TestHTTPProvider_Submit_UnreachableServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	p := NewHTTPProvider(url, &fakeStorage{})
	if _, err := p.Submit(context.Background(), domain.VerificationRequest{CaseID: uuid.New(), ApplicantID: uuid.New()}); err == nil {
		t.Fatal("Submit() error = nil, want error for unreachable server")
	}
}
