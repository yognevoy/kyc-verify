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

func (r *VerificationCaseRepository) Create(ctx context.Context, c *domain.VerificationCase, event *domain.VerificationCaseEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertCaseQ = `INSERT INTO verification_cases (id, applicant_id, status) VALUES ($1, $2, $3)
		RETURNING created_at, updated_at`
	if err := tx.QueryRow(ctx, insertCaseQ, c.ID, c.ApplicantID, c.Status).Scan(&c.CreatedAt, &c.UpdatedAt); err != nil {
		return fmt.Errorf("insert verification case: %w", err)
	}

	if err := insertEvent(ctx, tx, event); err != nil {
		return err
	}

	return tx.Commit(ctx)
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

func (r *VerificationCaseRepository) Transition(ctx context.Context, c *domain.VerificationCase, event *domain.VerificationCaseEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const updateQ = `UPDATE verification_cases SET status = $1, updated_at = now()
		WHERE id = $2 AND status = $3 RETURNING updated_at`
	err = tx.QueryRow(ctx, updateQ, c.Status, c.ID, event.FromStatus).Scan(&c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInvalidTransition
		}
		return fmt.Errorf("update verification case status: %w", err)
	}

	if err := insertEvent(ctx, tx, event); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *VerificationCaseRepository) ListQueue(ctx context.Context) ([]domain.QueueItem, error) {
	const q = `SELECT vc.id, vc.applicant_id, a.full_name, vc.status, vc.created_at, vc.updated_at
		FROM verification_cases vc
		JOIN applicants a ON a.id = vc.applicant_id
		WHERE vc.status IN ('submitted', 'in_review')
		ORDER BY vc.created_at`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query queue: %w", err)
	}
	defer rows.Close()

	var items []domain.QueueItem
	for rows.Next() {
		var item domain.QueueItem
		err := rows.Scan(&item.CaseID, &item.ApplicantID, &item.ApplicantFullName, &item.Status, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan queue item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate queue: %w", err)
	}
	return items, nil
}

func insertEvent(ctx context.Context, tx pgx.Tx, event *domain.VerificationCaseEvent) error {
	const q = `INSERT INTO verification_case_events (id, case_id, from_status, to_status, actor_type, actor_id, comment)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING created_at`

	err := tx.QueryRow(ctx, q, event.ID, event.CaseID, event.FromStatus, event.ToStatus, event.ActorType, event.ActorID, event.Comment).
		Scan(&event.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert verification case event: %w", err)
	}
	return nil
}
