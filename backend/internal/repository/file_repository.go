package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloudigo/backend/internal/model"
)

type FileRepository struct {
	pool *pgxpool.Pool
}

func NewFileRepository(pool *pgxpool.Pool) *FileRepository {
	return &FileRepository{pool: pool}
}

const fileColumns = `id, upload_id, secret_code, file_name, original_path, size_bytes, has_thumbnail, created_at`

func scanFile(row pgx.Row) (*model.File, error) {
	var f model.File
	if err := row.Scan(&f.ID, &f.UploadID, &f.SecretCode, &f.FileName, &f.OriginalPath, &f.SizeBytes, &f.HasThumbnail, &f.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

// Add inserts a file row using a caller-supplied id. The id is the same UUID
// the client generated when it started uploading the file, which doubles as
// the on-disk temp-file key — so Complete() can always find a file's
// temp-storage path from its DB row alone.
func (r *FileRepository) Add(ctx context.Context, id, uploadID uuid.UUID, secretCode, fileName string, size int64) (*model.File, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO files (id, upload_id, secret_code, file_name, size_bytes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+fileColumns,
		id, uploadID, secretCode, fileName, size,
	)
	return scanFile(row)
}

func (r *FileRepository) MarkThumbnail(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE files SET has_thumbnail = true WHERE id = $1`, id)
	return err
}

func (r *FileRepository) ListByUploadID(ctx context.Context, uploadID uuid.UUID) ([]model.File, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+fileColumns+` FROM files WHERE upload_id = $1 ORDER BY created_at`, uploadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.File
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, *f)
	}
	return files, rows.Err()
}

func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.File, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+fileColumns+` FROM files WHERE id = $1`, id)
	return scanFile(row)
}
