package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"kyc-verify/internal/domain"
)

type HTTPProvider struct {
	url     string
	storage domain.DocumentStorage
	client  *http.Client
}

func NewHTTPProvider(url string, storage domain.DocumentStorage) *HTTPProvider {
	return &HTTPProvider{url: url, storage: storage, client: &http.Client{Timeout: 60 * time.Second}}
}

type httpSubmitResponse struct {
	Reference string `json:"reference"`
}

func (p *HTTPProvider) Submit(ctx context.Context, req domain.VerificationRequest) (string, error) {
	pr, pw := io.Pipe()
	defer pr.Close()
	mw := multipart.NewWriter(pw)
	go func() {
		pw.CloseWithError(p.writeMultipart(ctx, mw, req))
	}()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, pr)
	if err != nil {
		return "", fmt.Errorf("build submit request: %w", err)
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())

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

func (p *HTTPProvider) writeMultipart(ctx context.Context, mw *multipart.Writer, req domain.VerificationRequest) error {
	if err := mw.WriteField("case_id", req.CaseID.String()); err != nil {
		return fmt.Errorf("write case_id: %w", err)
	}
	if err := mw.WriteField("applicant_id", req.ApplicantID.String()); err != nil {
		return fmt.Errorf("write applicant_id: %w", err)
	}

	for _, d := range req.Documents {
		if err := p.writeFile(ctx, mw, d); err != nil {
			return err
		}
	}
	return mw.Close()
}

func (p *HTTPProvider) writeFile(ctx context.Context, mw *multipart.Writer, d domain.Document) error {
	f, err := p.storage.Open(ctx, d.FilePath)
	if err != nil {
		return fmt.Errorf("open document %s: %w", d.ID, err)
	}
	defer f.Close()

	part, err := mw.CreateFormFile(string(d.Type), d.ID.String()+filepath.Ext(d.FilePath))
	if err != nil {
		return fmt.Errorf("create file part for document %s: %w", d.ID, err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return fmt.Errorf("write file part for document %s: %w", d.ID, err)
	}
	return nil
}
