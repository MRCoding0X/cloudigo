package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloudigo/backend/internal/model"
)

type UploadRepository struct {
	pool *pgxpool.Pool
}

func NewUploadRepository(pool *pgxpool.Pool) *UploadRepository {
	return &UploadRepository{pool: pool}
}

const uploadColumns = `
	id, upload_id, secret_code, share_code, user_id, email_from, message, password_hash, destruct,
	share_type, status, encrypt_key, file_count, total_size_bytes,
	ip, created_at, expires_at`

func scanUpload(row pgx.Row) (*model.Upload, error) {
	var u model.Upload
	if err := row.Scan(
		&u.ID, &u.UploadID, &u.SecretCode, &u.ShareCode, &u.UserID, &u.EmailFrom, &u.Message, &u.PasswordHash, &u.Destruct,
		&u.ShareType, &u.Status, &u.EncryptKey, &u.FileCount, &u.TotalSizeBytes,
		&u.IP, &u.CreatedAt, &u.ExpiresAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// CreateSession creates a placeholder upload row (status=processing) so files
// can reference a real foreign key as soon as they finish uploading, before
// the sender has submitted the register() metadata. secretCode is the
// owner-only management key (never handed to recipients); shareCode is the
// separate, download-only code recipients actually receive.
func (r *UploadRepository) CreateSession(ctx context.Context, uploadID, secretCode, shareCode, ip string) (*model.Upload, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO uploads (upload_id, secret_code, share_code, ip, status)
		VALUES ($1, $2, $3, $4, 'processing')
		RETURNING `+uploadColumns,
		uploadID, secretCode, shareCode, ip,
	)
	return scanUpload(row)
}

func (r *UploadRepository) GetByUploadID(ctx context.Context, uploadID string) (*model.Upload, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+uploadColumns+` FROM uploads WHERE upload_id = $1`, uploadID)
	return scanUpload(row)
}

func (r *UploadRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Upload, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+uploadColumns+` FROM uploads WHERE id = $1`, id)
	return scanUpload(row)
}

type RegisterParams struct {
	EmailFrom    string
	Message      string
	PasswordHash *string
	Destruct     bool
	ShareType    string
	ExpiresAt    *time.Time
}

func (r *UploadRepository) Register(ctx context.Context, id uuid.UUID, p RegisterParams) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE uploads
		SET email_from = $2, message = $3, password_hash = $4, destruct = $5,
		    share_type = $6, expires_at = $7
		WHERE id = $1`,
		id, p.EmailFrom, p.Message, p.PasswordHash, p.Destruct, p.ShareType, p.ExpiresAt,
	)
	return err
}

// UpdateOwnerSettings lets an uploader change their password/expiry after
// the upload is already ready — a passwordHash of nil clears the password.
func (r *UploadRepository) UpdateOwnerSettings(ctx context.Context, id uuid.UUID, passwordHash *string, expiresAt *time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE uploads SET password_hash = $2, expires_at = $3 WHERE id = $1`,
		id, passwordHash, expiresAt,
	)
	return err
}

func (r *UploadRepository) MarkReady(ctx context.Context, id uuid.UUID, fileCount int, totalSize int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE uploads SET status = 'ready', file_count = $2, total_size_bytes = $3 WHERE id = $1`,
		id, fileCount, totalSize,
	)
	return err
}

func (r *UploadRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE uploads SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (r *UploadRepository) StoreEncryptKey(ctx context.Context, id uuid.UUID, key string) error {
	_, err := r.pool.Exec(ctx, `UPDATE uploads SET encrypt_key = $2 WHERE id = $1`, id, key)
	return err
}

// GetExpired returns uploads past their expiry that haven't been cleaned up yet.
func (r *UploadRepository) GetExpired(ctx context.Context) ([]model.Upload, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+uploadColumns+` FROM uploads
		WHERE status IN ('ready', 'inactive') AND expires_at IS NOT NULL AND expires_at < now()`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectUploads(rows)
}

// GetStuckProcessing returns uploads that never completed (browser closed
// mid-upload, complete() never called) older than the given age.
func (r *UploadRepository) GetStuckProcessing(ctx context.Context, olderThan time.Duration) ([]model.Upload, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+uploadColumns+` FROM uploads
		WHERE status = 'processing' AND created_at < $1`,
		time.Now().Add(-olderThan),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectUploads(rows)
}

func collectUploads(rows pgx.Rows) ([]model.Upload, error) {
	var uploads []model.Upload
	for rows.Next() {
		u, err := scanUpload(rows)
		if err != nil {
			return nil, err
		}
		uploads = append(uploads, *u)
	}
	return uploads, rows.Err()
}

// Delete permanently removes an upload row (and, via ON DELETE CASCADE, its
// files/receivers/downloads) — used for uploads that never became visible
// (stuck in "processing"), so there is nothing worth auditing.
func (r *UploadRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM uploads WHERE id = $1`, id)
	return err
}

func (r *UploadRepository) CountByIPSince(ctx context.Context, ip string, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM uploads WHERE ip = $1 AND created_at >= $2`,
		ip, since,
	).Scan(&count)
	return count, err
}

type AdminUploadFilter struct {
	Status string // "" = any
	Search string // matches upload_id or email_from (case-insensitive substring)
}

// ListForAdmin powers the admin uploads table: paginated, optionally
// filtered by status and a free-text search over upload_id/email_from.
func (r *UploadRepository) ListForAdmin(ctx context.Context, filter AdminUploadFilter, offset, limit int) ([]model.Upload, int, error) {
	where := "WHERE ($1 = '' OR status = $1) AND ($2 = '' OR upload_id ILIKE '%'||$2||'%' OR email_from ILIKE '%'||$2||'%')"

	rows, err := r.pool.Query(ctx, `
		SELECT `+uploadColumns+` FROM uploads `+where+`
		ORDER BY created_at DESC OFFSET $3 LIMIT $4`,
		filter.Status, filter.Search, offset, limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	uploads, err := collectUploads(rows)
	if err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM uploads `+where, filter.Status, filter.Search).Scan(&total); err != nil {
		return nil, 0, err
	}
	return uploads, total, nil
}

// ListAllForExport returns every upload matching filter, unpaginated (CSV export).
func (r *UploadRepository) ListAllForExport(ctx context.Context, filter AdminUploadFilter) ([]model.Upload, error) {
	where := "WHERE ($1 = '' OR status = $1) AND ($2 = '' OR upload_id ILIKE '%'||$2||'%' OR email_from ILIKE '%'||$2||'%')"
	rows, err := r.pool.Query(ctx, `SELECT `+uploadColumns+` FROM uploads `+where+` ORDER BY created_at DESC`, filter.Status, filter.Search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectUploads(rows)
}

type DashboardStats struct {
	TotalUploads      int
	ActiveUploads     int
	DestroyedUploads  int
	TotalDownloads    int
	TotalStorageBytes int64
}

func (r *UploadRepository) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	var s DashboardStats
	err := r.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM uploads),
			(SELECT COUNT(*) FROM uploads WHERE status = 'ready'),
			(SELECT COUNT(*) FROM uploads WHERE status = 'destroyed'),
			(SELECT COUNT(*) FROM downloads),
			(SELECT COALESCE(SUM(total_size_bytes), 0) FROM uploads WHERE status = 'ready')
	`).Scan(&s.TotalUploads, &s.ActiveUploads, &s.DestroyedUploads, &s.TotalDownloads, &s.TotalStorageBytes)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

type DailyStat struct {
	Day          string `json:"day"`
	Uploads      int    `json:"uploads"`
	Downloads    int    `json:"downloads"`
	StorageBytes int64  `json:"storageBytes"`
}

// GetDailyStats returns one row per calendar day for the last `days` days
// (including today), left-joining upload/download activity so days with no
// activity still appear with zero counts — needed for a gap-free chart.
func (r *UploadRepository) GetDailyStats(ctx context.Context, days int) ([]DailyStat, error) {
	rows, err := r.pool.Query(ctx, `
		WITH bounds AS (
			SELECT (CURRENT_DATE - ($1::int - 1)) AS start_day
		),
		day_series AS (
			SELECT generate_series(start_day, CURRENT_DATE, interval '1 day')::date AS day FROM bounds
		),
		upload_stats AS (
			SELECT created_at::date AS day, COUNT(*) AS uploads, COALESCE(SUM(total_size_bytes), 0) AS bytes
			FROM uploads, bounds
			WHERE created_at::date >= start_day
			GROUP BY 1
		),
		download_stats AS (
			SELECT downloaded_at::date AS day, COUNT(*) AS downloads
			FROM downloads, bounds
			WHERE downloaded_at::date >= start_day
			GROUP BY 1
		)
		SELECT to_char(d.day, 'YYYY-MM-DD'), COALESCE(u.uploads, 0), COALESCE(dl.downloads, 0), COALESCE(u.bytes, 0)
		FROM day_series d
		LEFT JOIN upload_stats u ON u.day = d.day
		LEFT JOIN download_stats dl ON dl.day = d.day
		ORDER BY d.day`,
		days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStat
	for rows.Next() {
		var s DailyStat
		if err := rows.Scan(&s.Day, &s.Uploads, &s.Downloads, &s.StorageBytes); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
