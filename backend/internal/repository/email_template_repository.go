package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloudigo/backend/internal/model"
)

type EmailTemplateRepository struct {
	pool *pgxpool.Pool
}

func NewEmailTemplateRepository(pool *pgxpool.Pool) *EmailTemplateRepository {
	return &EmailTemplateRepository{pool: pool}
}

const emailTemplateColumns = `id, type, lang, subject, body, enabled`

func scanEmailTemplate(row pgx.Row) (*model.EmailTemplate, error) {
	var t model.EmailTemplate
	if err := row.Scan(&t.ID, &t.Type, &t.Lang, &t.Subject, &t.Body, &t.Enabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// GetByTypeAndLang falls back to English if the requested language has no
// template for this type yet (languages beyond English are a Fas 5 concern).
func (r *EmailTemplateRepository) GetByTypeAndLang(ctx context.Context, templateType, lang string) (*model.EmailTemplate, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+emailTemplateColumns+` FROM email_templates WHERE type = $1 AND lang = $2`, templateType, lang)
	t, err := scanEmailTemplate(row)
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, ErrNotFound) || lang == "en" {
		return nil, err
	}

	row = r.pool.QueryRow(ctx, `SELECT `+emailTemplateColumns+` FROM email_templates WHERE type = $1 AND lang = 'en'`, templateType)
	return scanEmailTemplate(row)
}

// List returns every template row (all types/langs) for the admin editor,
// ordered so each type's translations sit together.
func (r *EmailTemplateRepository) List(ctx context.Context) ([]model.EmailTemplate, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+emailTemplateColumns+` FROM email_templates ORDER BY type, lang`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.EmailTemplate
	for rows.Next() {
		t, err := scanEmailTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// Upsert saves a template keyed by (type, lang) — the admin editor can either
// update an existing translation or add a new language for an existing type,
// since email_templates has no separate create/delete flow (its type set is
// fixed by a CHECK constraint; see migrations).
func (r *EmailTemplateRepository) Upsert(ctx context.Context, t *model.EmailTemplate) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO email_templates (type, lang, subject, body, enabled)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (type, lang) DO UPDATE SET
			subject = EXCLUDED.subject, body = EXCLUDED.body, enabled = EXCLUDED.enabled`,
		t.Type, t.Lang, t.Subject, t.Body, t.Enabled,
	)
	return err
}
