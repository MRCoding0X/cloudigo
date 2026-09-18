package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloudigo/backend/internal/model"
)

type BackgroundRepository struct {
	pool *pgxpool.Pool
}

func NewBackgroundRepository(pool *pgxpool.Pool) *BackgroundRepository {
	return &BackgroundRepository{pool: pool}
}

const backgroundColumns = `id, src, url, duration_seconds, sort_order, created_at`

func scanBackground(row pgx.Row) (*model.Background, error) {
	var b model.Background
	if err := row.Scan(&b.ID, &b.Src, &b.URL, &b.DurationSeconds, &b.SortOrder, &b.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (r *BackgroundRepository) List(ctx context.Context) ([]model.Background, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+backgroundColumns+` FROM backgrounds ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Background
	for rows.Next() {
		b, err := scanBackground(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *b)
	}
	return list, rows.Err()
}

func (r *BackgroundRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Background, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+backgroundColumns+` FROM backgrounds WHERE id = $1`, id)
	return scanBackground(row)
}

// Create takes an explicit id (rather than DEFAULT gen_random_uuid()) so the
// caller can compute the on-disk file path — which is derived from
// (id, original filename) — before the row exists.
func (r *BackgroundRepository) Create(ctx context.Context, id uuid.UUID, src, url string, duration *int) (*model.Background, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO backgrounds (id, src, url, duration_seconds) VALUES ($1, $2, $3, $4)
		RETURNING `+backgroundColumns,
		id, src, url, duration,
	)
	return scanBackground(row)
}

func (r *BackgroundRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM backgrounds WHERE id = $1`, id)
	return err
}
