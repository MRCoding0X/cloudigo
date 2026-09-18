package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloudigo/backend/internal/model"
)

type BlockedIPRepository struct {
	pool *pgxpool.Pool
}

func NewBlockedIPRepository(pool *pgxpool.Pool) *BlockedIPRepository {
	return &BlockedIPRepository{pool: pool}
}

func (r *BlockedIPRepository) List(ctx context.Context) ([]model.BlockedIP, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, ip, reason, created_at FROM blocked_ips ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.BlockedIP
	for rows.Next() {
		var b model.BlockedIP
		if err := rows.Scan(&b.ID, &b.IP, &b.Reason, &b.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (r *BlockedIPRepository) Add(ctx context.Context, ip, reason string) (*model.BlockedIP, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO blocked_ips (ip, reason) VALUES ($1, $2)
		ON CONFLICT (ip) DO UPDATE SET reason = EXCLUDED.reason
		RETURNING id, ip, reason, created_at`,
		ip, reason,
	)
	var b model.BlockedIP
	if err := row.Scan(&b.ID, &b.IP, &b.Reason, &b.CreatedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BlockedIPRepository) Remove(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocked_ips WHERE id = $1`, id)
	return err
}

// IsBlocked reports whether ip is on the blocklist. Cheap indexed lookup
// (unique index on ip via the UNIQUE constraint), called on every upload
// session creation.
func (r *BlockedIPRepository) IsBlocked(ctx context.Context, ip string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM blocked_ips WHERE ip = $1)`, ip).Scan(&exists)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return exists, nil
}
