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

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

const userColumns = `id, email, password_hash, role, ip, reset_token, reset_token_expires_at, created_at, updated_at`

func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	if err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IP,
		&u.ResetToken, &u.ResetTokenExpiresAt, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash, role, ip string) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role, ip)
		VALUES ($1, $2, $3, $4)
		RETURNING `+userColumns,
		email, passwordHash, role, ip,
	)
	return scanUser(row)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (r *UserRepository) GetByResetToken(ctx context.Context, token string) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE reset_token = $1`, token)
	return scanUser(row)
}

func (r *UserRepository) UpdateIP(ctx context.Context, id uuid.UUID, ip string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET ip = $2, updated_at = now() WHERE id = $1`, id, ip)
	return err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, reset_token = NULL, reset_token_expires_at = NULL, updated_at = now()
		WHERE id = $1`,
		id, passwordHash,
	)
	return err
}

func (r *UserRepository) SetResetToken(ctx context.Context, id uuid.UUID, token string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET reset_token = $2, reset_token_expires_at = $3, updated_at = now() WHERE id = $1`,
		id, token, expiresAt,
	)
	return err
}

// List paginates users, optionally filtered by a case-insensitive substring
// match on email ("" means no filter).
func (r *UserRepository) List(ctx context.Context, search string, offset, limit int) ([]model.User, int, error) {
	where := `WHERE ($1 = '' OR email ILIKE '%'||$1||'%')`

	rows, err := r.pool.Query(ctx, `
		SELECT `+userColumns+` FROM users `+where+`
		ORDER BY created_at DESC OFFSET $2 LIMIT $3`,
		search, offset, limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users `+where, search).Scan(&total); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// UpdateEmail changes only the email — used for self-service account
// updates, where the caller must never be able to change their own role.
func (r *UserRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE users SET email = $2, updated_at = now() WHERE id = $1
		RETURNING `+userColumns,
		id, email,
	)
	return scanUser(row)
}

func (r *UserRepository) UpdateEmailAndRole(ctx context.Context, id uuid.UUID, email, role string) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE users SET email = $2, role = $3, updated_at = now() WHERE id = $1
		RETURNING `+userColumns,
		id, email, role,
	)
	return scanUser(row)
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
