package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

type User struct {
	ID                  uuid.UUID
	Email               string
	PasswordHash        string
	Role                string
	IP                  string
	ResetToken          *string
	ResetTokenExpiresAt *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
