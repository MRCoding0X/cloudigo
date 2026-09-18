package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DownloadRepository struct {
	pool *pgxpool.Pool
}

func NewDownloadRepository(pool *pgxpool.Pool) *DownloadRepository {
	return &DownloadRepository{pool: pool}
}

func (r *DownloadRepository) Insert(ctx context.Context, uploadID uuid.UUID, email, ip string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO downloads (upload_id, email, ip) VALUES ($1, $2, $3)`, uploadID, email, ip)
	return err
}

func (r *DownloadRepository) CountByUploadID(ctx context.Context, uploadID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM downloads WHERE upload_id = $1`, uploadID).Scan(&count)
	return count, err
}

type AdminDownloadRow struct {
	UploadID     string
	Email        string
	IP           string
	DownloadedAt string
}

// ListForAdmin joins in the public upload_id string (the admin UI shows that,
// not the internal UUID) for the downloads log table. search matches
// upload_id, downloader email, or IP (case-insensitive substring); "" means
// no filter.
func (r *DownloadRepository) ListForAdmin(ctx context.Context, search string, offset, limit int) ([]AdminDownloadRow, int, error) {
	where := `WHERE ($1 = '' OR u.upload_id ILIKE '%'||$1||'%' OR d.email ILIKE '%'||$1||'%' OR d.ip ILIKE '%'||$1||'%')`

	rows, err := r.pool.Query(ctx, `
		SELECT u.upload_id, d.email, d.ip, to_char(d.downloaded_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM downloads d
		JOIN uploads u ON u.id = d.upload_id
		`+where+`
		ORDER BY d.downloaded_at DESC
		OFFSET $2 LIMIT $3`,
		search, offset, limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []AdminDownloadRow
	for rows.Next() {
		var row AdminDownloadRow
		if err := rows.Scan(&row.UploadID, &row.Email, &row.IP, &row.DownloadedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM downloads d JOIN uploads u ON u.id = d.upload_id `+where,
		search,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListAllForExport returns every download log row matching search,
// unpaginated (CSV export) — same filter as ListForAdmin.
func (r *DownloadRepository) ListAllForExport(ctx context.Context, search string) ([]AdminDownloadRow, error) {
	where := `WHERE ($1 = '' OR u.upload_id ILIKE '%'||$1||'%' OR d.email ILIKE '%'||$1||'%' OR d.ip ILIKE '%'||$1||'%')`
	rows, err := r.pool.Query(ctx, `
		SELECT u.upload_id, d.email, d.ip, to_char(d.downloaded_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM downloads d
		JOIN uploads u ON u.id = d.upload_id
		`+where+`
		ORDER BY d.downloaded_at DESC`,
		search,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AdminDownloadRow
	for rows.Next() {
		var row AdminDownloadRow
		if err := rows.Scan(&row.UploadID, &row.Email, &row.IP, &row.DownloadedAt); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// HasAllReceiversDownloaded reports whether every receiver's email address
// appears at least once in the download log for this upload.
func (r *DownloadRepository) HasAllReceiversDownloaded(ctx context.Context, uploadID uuid.UUID) (bool, error) {
	var missing int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM receivers rcv
		WHERE rcv.upload_id = $1
		  AND NOT EXISTS (
		    SELECT 1 FROM downloads d WHERE d.upload_id = rcv.upload_id AND d.email = rcv.email
		  )`,
		uploadID,
	).Scan(&missing)
	if err != nil {
		return false, err
	}
	return missing == 0, nil
}
