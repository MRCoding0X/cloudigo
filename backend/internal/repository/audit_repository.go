package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"cloudigo/backend/internal/model"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Insert(ctx context.Context, actorEmail, eventType, details, ip string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_log (actor_email, event_type, details, ip) VALUES ($1, $2, $3, $4)`,
		actorEmail, eventType, details, ip,
	)
	return err
}

const auditSearchWhere = `WHERE ($1 = '' OR actor_email ILIKE '%'||$1||'%' OR event_type ILIKE '%'||$1||'%' OR details ILIKE '%'||$1||'%' OR ip ILIKE '%'||$1||'%')`

// List paginates the audit log, optionally filtered by a case-insensitive
// substring match on actor email, event type, details, or IP ("" means no filter).
func (r *AuditRepository) List(ctx context.Context, search string, offset, limit int) ([]model.AuditEntry, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, actor_email, event_type, details, ip, created_at
		FROM audit_log `+auditSearchWhere+`
		ORDER BY created_at DESC OFFSET $2 LIMIT $3`,
		search, offset, limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []model.AuditEntry
	for rows.Next() {
		var e model.AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorEmail, &e.EventType, &e.Details, &e.IP, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_log `+auditSearchWhere, search).Scan(&total); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

// ListAllForExport returns every audit log row matching search, unpaginated (CSV export).
func (r *AuditRepository) ListAllForExport(ctx context.Context, search string) ([]model.AuditEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, actor_email, event_type, details, ip, created_at
		FROM audit_log `+auditSearchWhere+`
		ORDER BY created_at DESC`,
		search,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.AuditEntry
	for rows.Next() {
		var e model.AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorEmail, &e.EventType, &e.Details, &e.IP, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
