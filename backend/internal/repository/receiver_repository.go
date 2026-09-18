package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloudigo/backend/internal/model"
)

type ReceiverRepository struct {
	pool *pgxpool.Pool
}

func NewReceiverRepository(pool *pgxpool.Pool) *ReceiverRepository {
	return &ReceiverRepository{pool: pool}
}

const receiverColumns = `id, upload_id, email, private_id, created_at`

func scanReceiver(row pgx.Row) (*model.Receiver, error) {
	var rcv model.Receiver
	if err := row.Scan(&rcv.ID, &rcv.UploadID, &rcv.Email, &rcv.PrivateID, &rcv.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rcv, nil
}

func (r *ReceiverRepository) Add(ctx context.Context, uploadID uuid.UUID, email, privateID string) (*model.Receiver, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO receivers (upload_id, email, private_id) VALUES ($1, $2, $3)
		RETURNING `+receiverColumns,
		uploadID, email, privateID,
	)
	return scanReceiver(row)
}

func (r *ReceiverRepository) ListByUploadID(ctx context.Context, uploadID uuid.UUID) ([]model.Receiver, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+receiverColumns+` FROM receivers WHERE upload_id = $1`, uploadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Receiver
	for rows.Next() {
		rcv, err := scanReceiver(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *rcv)
	}
	return list, rows.Err()
}

func (r *ReceiverRepository) GetByUploadAndPrivateID(ctx context.Context, uploadID uuid.UUID, privateID string) (*model.Receiver, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+receiverColumns+` FROM receivers WHERE upload_id = $1 AND private_id = $2`, uploadID, privateID)
	return scanReceiver(row)
}
