package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"kyc-verify/internal/domain"
)

type ApplicantRepository struct {
	pool *pgxpool.Pool
}

func NewApplicantRepository(pool *pgxpool.Pool) *ApplicantRepository {
	return &ApplicantRepository{pool: pool}
}

func (r *ApplicantRepository) Create(ctx context.Context, applicant *domain.Applicant) error {
	const q = `INSERT INTO applicants (id, user_id, full_name, birth_date, country, risk_level)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, q,
		applicant.ID, applicant.UserID, applicant.FullName, applicant.BirthDate, applicant.Country, applicant.RiskLevel)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return domain.ErrApplicantAlreadyExists
		}
		return fmt.Errorf("insert applicant: %w", err)
	}
	return nil
}

func (r *ApplicantRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Applicant, error) {
	const q = `SELECT id, user_id, full_name, birth_date, country, risk_level, created_at, updated_at
		FROM applicants WHERE user_id = $1`

	var a domain.Applicant
	err := r.pool.QueryRow(ctx, q, userID).
		Scan(&a.ID, &a.UserID, &a.FullName, &a.BirthDate, &a.Country, &a.RiskLevel, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrApplicantNotFound
		}
		return nil, fmt.Errorf("query applicant: %w", err)
	}
	return &a, nil
}

func (r *ApplicantRepository) Update(ctx context.Context, applicant *domain.Applicant) error {
	const q = `UPDATE applicants SET full_name = $1, birth_date = $2, country = $3, updated_at = now()
		WHERE id = $4`

	if _, err := r.pool.Exec(ctx, q, applicant.FullName, applicant.BirthDate, applicant.Country, applicant.ID); err != nil {
		return fmt.Errorf("update applicant: %w", err)
	}
	return nil
}
