package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kyc-verify/internal/domain"
)

type VerificationCaseRepository struct {
	pool *pgxpool.Pool
}

func NewVerificationCaseRepository(pool *pgxpool.Pool) *VerificationCaseRepository {
	return &VerificationCaseRepository{pool: pool}
}

func (r *VerificationCaseRepository) Create(ctx context.Context, c *domain.VerificationCase) error {
	const q = `INSERT INTO verification_cases (id, applicant_id, status) VALUES ($1, $2, $3)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, q, c.ID, c.ApplicantID, c.Status).Scan(&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert verification case: %w", err)
	}
	return nil
}

func (r *VerificationCaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.VerificationCase, error) {
	const q = `SELECT id, applicant_id, status, created_at, updated_at FROM verification_cases WHERE id = $1`

	var c domain.VerificationCase
	err := r.pool.QueryRow(ctx, q, id).Scan(&c.ID, &c.ApplicantID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVerificationCaseNotFound
		}
		return nil, fmt.Errorf("query verification case: %w", err)
	}
	return &c, nil
}

func (r *VerificationCaseRepository) GetLatestByApplicantID(ctx context.Context, applicantID uuid.UUID) (*domain.VerificationCase, error) {
	const q = `SELECT id, applicant_id, status, created_at, updated_at FROM verification_cases
		WHERE applicant_id = $1 ORDER BY created_at DESC LIMIT 1`

	var c domain.VerificationCase
	err := r.pool.QueryRow(ctx, q, applicantID).Scan(&c.ID, &c.ApplicantID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVerificationCaseNotFound
		}
		return nil, fmt.Errorf("query latest verification case: %w", err)
	}
	return &c, nil
}

func (r *VerificationCaseRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.CaseStatus) error {
	const q = `UPDATE verification_cases SET status = $1, updated_at = now() WHERE id = $2`

	if _, err := r.pool.Exec(ctx, q, status, id); err != nil {
		return fmt.Errorf("update verification case status: %w", err)
	}
	return nil
}
