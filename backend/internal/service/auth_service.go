package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"cloudigo/backend/internal/model"
	"cloudigo/backend/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrEmailTaken         = errors.New("that email is already in use")
)

type AuthService struct {
	users         *repository.UserRepository
	refreshTokens *repository.RefreshTokenRepository
	email         *EmailService
	frontendURL   string

	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewAuthService(
	users *repository.UserRepository,
	refreshTokens *repository.RefreshTokenRepository,
	email *EmailService,
	frontendURL string,
	accessSecret, refreshSecret string,
	accessTTLMin, refreshTTLDays int,
) *AuthService {
	return &AuthService{
		users:         users,
		refreshTokens: refreshTokens,
		email:         email,
		frontendURL:   frontendURL,
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     time.Duration(accessTTLMin) * time.Minute,
		refreshTTL:    time.Duration(refreshTTLDays) * 24 * time.Hour,
	}
}

type AccessClaims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type AuthResult struct {
	User                  *model.User
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

// --- password hashing ---

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// --- tokens ---

func randomToken(numBytes int) (string, error) {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *AuthService) issueAccessToken(u *model.User) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		UserID: u.ID.String(),
		Role:   u.Role,
		Email:  u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			Subject:   u.ID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.accessSecret)
}

// ParseAccessToken validates an access token and returns its claims.
func (s *AuthService) ParseAccessToken(tokenStr string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.accessSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (s *AuthService) issueRefreshToken(ctx context.Context, u *model.User) (string, time.Time, error) {
	raw, err := randomToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(s.refreshTTL)

	if _, err := s.refreshTokens.Create(ctx, u.ID, hashToken(raw), expiresAt); err != nil {
		return "", time.Time{}, err
	}

	return raw, expiresAt, nil
}

// --- public operations ---

func (s *AuthService) Login(ctx context.Context, email, password, ip string) (*AuthResult, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !checkPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	_ = s.users.UpdateIP(ctx, user.ID, ip)

	access, err := s.issueAccessToken(user)
	if err != nil {
		return nil, err
	}
	refresh, refreshExp, err := s.issueRefreshToken(ctx, user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:                  user,
		AccessToken:           access,
		RefreshToken:          refresh,
		RefreshTokenExpiresAt: refreshExp,
	}, nil
}

// Refresh rotates a refresh token: the old one is revoked and a new access+refresh pair is issued.
func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*AuthResult, error) {
	stored, err := s.refreshTokens.GetByHash(ctx, hashToken(rawRefreshToken))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	if stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if err := s.refreshTokens.Revoke(ctx, stored.ID); err != nil {
		return nil, err
	}

	access, err := s.issueAccessToken(user)
	if err != nil {
		return nil, err
	}
	refresh, refreshExp, err := s.issueRefreshToken(ctx, user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:                  user,
		AccessToken:           access,
		RefreshToken:          refresh,
		RefreshTokenExpiresAt: refreshExp,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	stored, err := s.refreshTokens.GetByHash(ctx, hashToken(rawRefreshToken))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	return s.refreshTokens.Revoke(ctx, stored.ID)
}

// LogoutOtherSessions revokes every session for userID except the one tied
// to rawRefreshToken (the caller's own), returning how many were revoked.
func (s *AuthService) LogoutOtherSessions(ctx context.Context, userID uuid.UUID, rawRefreshToken string) (int, error) {
	return s.refreshTokens.RevokeAllForUserExcept(ctx, userID, hashToken(rawRefreshToken))
}

// ForgotPassword issues a password reset token (valid 1 hour) if the email
// exists and belongs to a non-admin account. Admin passwords are never
// resettable via email — a compromised inbox must not be able to take over
// the admin account — so an admin email is treated exactly like an unknown
// one. It never reveals whether the email exists (or is an admin) to the caller.
func (s *AuthService) ForgotPassword(ctx context.Context, email string) (token string, user *model.User, err error) {
	user, err = s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", nil, nil
		}
		return "", nil, err
	}
	if user.Role == model.RoleAdmin {
		return "", nil, nil
	}

	raw, genErr := randomToken(32)
	if genErr != nil {
		return "", nil, genErr
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	if err := s.users.SetResetToken(ctx, user.ID, raw, expiresAt); err != nil {
		return "", nil, err
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, raw)
	go func() {
		if err := s.email.Send(context.Background(), user.Email, "password_reset", "en", map[string]string{
			"reset_url": resetURL,
		}); err != nil {
			log.Printf("send password reset email to %s: %v", user.Email, err)
		}
	}()

	return raw, user, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, err := s.users.GetByResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidToken
		}
		return err
	}

	if user.ResetTokenExpiresAt == nil || time.Now().After(*user.ResetTokenExpiresAt) {
		return ErrInvalidToken
	}
	if user.Role == model.RoleAdmin {
		return ErrInvalidToken
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.users.UpdatePassword(ctx, user.ID, hash); err != nil {
		return err
	}

	// Reset invalidates all existing sessions.
	return s.refreshTokens.RevokeAllForUser(ctx, user.ID)
}

func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.users.GetByID(ctx, id)
}

// UpdateOwnProfile lets a logged-in user change their own email and/or
// password (either can be left blank to leave it unchanged), requiring their
// current password first — self-service, so it never touches role, unlike
// the admin-users CRUD path.
func (s *AuthService) UpdateOwnProfile(ctx context.Context, id uuid.UUID, currentPassword, newEmail, newPassword string) (*model.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !checkPassword(user.PasswordHash, currentPassword) {
		return nil, ErrInvalidCredentials
	}

	if newEmail != "" && newEmail != user.Email {
		if existing, err := s.users.GetByEmail(ctx, newEmail); err == nil && existing.ID != user.ID {
			return nil, ErrEmailTaken
		} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		user, err = s.users.UpdateEmail(ctx, id, newEmail)
		if err != nil {
			return nil, err
		}
	}

	if newPassword != "" {
		hash, err := HashPassword(newPassword)
		if err != nil {
			return nil, err
		}
		// Deliberately does not revoke other sessions the way the
		// forgot-password reset flow does: that flow is an out-of-band
		// recovery path with no "current session" to preserve, while this is
		// an already-authenticated admin updating their own password and
		// should stay logged in on this device afterward.
		if err := s.users.UpdatePassword(ctx, id, hash); err != nil {
			return nil, err
		}
	}

	return user, nil
}

// --- short-lived download tickets ---
//
// A ticket lets the frontend turn a POST-verified password check into a
// browser-navigable GET download URL without ever putting the password
// itself in a URL or query string.

type DownloadTicketClaims struct {
	UploadID string `json:"uid"`
	Code     string `json:"code"`
	jwt.RegisteredClaims
}

func (s *AuthService) IssueDownloadTicket(uploadID, code string) (string, error) {
	now := time.Now()
	claims := DownloadTicketClaims{
		UploadID: uploadID,
		Code:     code,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.accessSecret)
}

func (s *AuthService) ParseDownloadTicket(tokenStr string) (*DownloadTicketClaims, error) {
	claims := &DownloadTicketClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.accessSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
