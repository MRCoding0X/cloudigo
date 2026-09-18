package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloudigo/backend/internal/model"
)

type PageRepository struct {
	pool *pgxpool.Pool
}

func NewPageRepository(pool *pgxpool.Pool) *PageRepository {
	return &PageRepository{pool: pool}
}

const pageColumns = `id, type, lang, title, content, sort_order, created_at, updated_at`

func scanPage(row pgx.Row) (*model.Page, error) {
	var p model.Page
	if err := row.Scan(&p.ID, &p.Type, &p.Lang, &p.Title, &p.Content, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *PageRepository) List(ctx context.Context) ([]model.Page, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+pageColumns+` FROM pages ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []model.Page
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		pages = append(pages, *p)
	}
	return pages, rows.Err()
}

func (r *PageRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Page, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+pageColumns+` FROM pages WHERE id = $1`, id)
	return scanPage(row)
}

func (r *PageRepository) GetByTypeAndLang(ctx context.Context, pageType, lang string) (*model.Page, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+pageColumns+` FROM pages WHERE type = $1 AND lang = $2`, pageType, lang)
	return scanPage(row)
}

func (r *PageRepository) Create(ctx context.Context, p *model.Page) (*model.Page, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO pages (type, lang, title, content, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+pageColumns,
		p.Type, p.Lang, p.Title, p.Content, p.SortOrder,
	)
	return scanPage(row)
}

func (r *PageRepository) Update(ctx context.Context, id uuid.UUID, p *model.Page) (*model.Page, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE pages SET type = $2, lang = $3, title = $4, content = $5, sort_order = $6, updated_at = now()
		WHERE id = $1
		RETURNING `+pageColumns,
		id, p.Type, p.Lang, p.Title, p.Content, p.SortOrder,
	)
	return scanPage(row)
}

func (r *PageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pages WHERE id = $1`, id)
	return err
}
