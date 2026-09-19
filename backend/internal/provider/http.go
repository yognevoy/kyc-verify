package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"kyc-verify/internal/domain"
)

type HTTPProvider struct {
	url    string
	client *http.Client
}

func NewHTTPProvider(url string) *HTTPProvider {
	return &HTTPProvider{url: url, client: &http.Client{Timeout: 10 * time.Second}}
}

type httpSubmitDocument struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type httpSubmitRequest struct {
	CaseID      string               `json:"case_id"`
	ApplicantID string               `json:"applicant_id"`
	Documents   []httpSubmitDocument `json:"documents"`
}

type httpSubmitResponse struct {
	Reference string `json:"reference"`
}

func (p *HTTPProvider) Submit(ctx context.Context, req domain.VerificationRequest) (string, error) {
	docs := make([]httpSubmitDocument, len(req.Documents))
	for i, d := range req.Documents {
		docs[i] = httpSubmitDocument{ID: d.ID.String(), Type: string(d.Type)}
	}

	body, err := json.Marshal(httpSubmitRequest{
		CaseID:      req.CaseID.String(),
		ApplicantID: req.ApplicantID.String(),
		Documents:   docs,
	})
	if err != nil {
		return "", fmt.Errorf("marshal submit request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build submit request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("submit request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("submit request: unexpected status %d", resp.StatusCode)
	}

	var out httpSubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode submit response: %w", err)
	}
	if out.Reference == "" {
		return "", fmt.Errorf("submit response: empty reference")
	}
	return out.Reference, nil
}
