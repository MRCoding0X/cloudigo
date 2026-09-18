package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailVerificationRepository struct {
	pool *pgxpool.Pool
}

func NewEmailVerificationRepository(pool *pgxpool.Pool) *EmailVerificationRepository {
	return &EmailVerificationRepository{pool: pool}
}

func (r *EmailVerificationRepository) Create(ctx context.Context, email, code string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO email_verifications (email, code, status) VALUES ($1, $2, 'pending')`, email, code)
	return err
}

// ConfirmPending marks the most recent pending code for this email verified,
// if it matches. Returns ErrNotFound if no matching pending code exists.
func (r *EmailVerificationRepository) ConfirmPending(ctx context.Context, email, code string, maxAge time.Duration) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE email_verifications
		SET status = 'verified'
		WHERE id = (
			SELECT id FROM email_verifications
			WHERE email = $1 AND code = $2 AND status = 'pending' AND created_at >= $3
			ORDER BY created_at DESC LIMIT 1
		)`,
		email, code, time.Now().Add(-maxAge),
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteOldPending removes stale, never-confirmed verification codes.
func (r *EmailVerificationRepository) DeleteOldPending(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM email_verifications WHERE status = 'pending' AND created_at < $1`,
		time.Now().Add(-olderThan),
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// HasVerified reports whether this email has a verified record within maxAge
// (used for email_verify="always") or ever (pass a very large maxAge, used
// for email_verify="once").
func (r *EmailVerificationRepository) HasVerified(ctx context.Context, email string, maxAge time.Duration) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM email_verifications
		WHERE email = $1 AND status = 'verified' AND created_at >= $2`,
		email, time.Now().Add(-maxAge),
	).Scan(&count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return count > 0, nil
}
