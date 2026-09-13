package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kyc-verify/internal/domain"
)

type DocumentRepository struct {
	pool *pgxpool.Pool
}

func NewDocumentRepository(pool *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{pool: pool}
}

func (r *DocumentRepository) Create(ctx context.Context, doc *domain.Document) error {
	const q = `INSERT INTO documents (id, applicant_id, type, file_path) VALUES ($1, $2, $3, $4) RETURNING uploaded_at`

	if err := r.pool.QueryRow(ctx, q, doc.ID, doc.ApplicantID, doc.Type, doc.FilePath).Scan(&doc.UploadedAt); err != nil {
		return fmt.Errorf("insert document: %w", err)
	}
	return nil
}

func (r *DocumentRepository) ListByApplicantID(ctx context.Context, applicantID uuid.UUID) ([]domain.Document, error) {
	const q = `SELECT id, applicant_id, type, file_path, uploaded_at FROM documents WHERE applicant_id = $1 ORDER BY uploaded_at`

	rows, err := r.pool.Query(ctx, q, applicantID)
	if err != nil {
		return nil, fmt.Errorf("query documents: %w", err)
	}
	defer rows.Close()

	var docs []domain.Document
	for rows.Next() {
		var d domain.Document
		if err := rows.Scan(&d.ID, &d.ApplicantID, &d.Type, &d.FilePath, &d.UploadedAt); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documents: %w", err)
	}
	return docs, nil
}
