package handler

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/google/uuid"

	"cloudigo/backend/internal/middleware"
	"cloudigo/backend/internal/service"
)

const refreshCookieName = "cloudigo_refresh"

type AuthHandler struct {
	auth          *service.AuthService
	audit         *service.AuditService
	secureCookies bool
}

func NewAuthHandler(auth *service.AuthService, audit *service.AuditService, secureCookies bool) *AuthHandler {
	return &AuthHandler{auth: auth, audit: audit, secureCookies: secureCookies}
}

// authRateLimit throttles brute-forceable auth endpoints per IP. In-memory
// (fine for this single-process deployment; a multi-instance deployment
// would need a shared store instead).
func authRateLimit(max int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many attempts, try again later")
		},
	})
}

func (h *AuthHandler) Register(router fiber.Router) {
	router.Post("/auth/login", authRateLimit(10), h.login)
	router.Post("/auth/refresh", h.refresh)
	router.Post("/auth/logout", h.logout)
	router.Post("/auth/forgot-password", authRateLimit(5), h.forgotPassword)
	router.Post("/auth/reset-password", authRateLimit(10), h.resetPassword)
	router.Get("/auth/me", middleware.RequireAuth(h.auth), h.me)
	router.Put("/auth/me", middleware.RequireAuth(h.auth), authRateLimit(10), h.updateMe)
	router.Post("/auth/logout-others", middleware.RequireAuth(h.auth), authRateLimit(10), h.logoutOthers)
}

func (h *AuthHandler) clearRefreshCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/auth",
		HTTPOnly: true,
		Secure:   h.secureCookies,
		SameSite: fiber.CookieSameSiteStrictMode,
		MaxAge:   -1,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil || req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}

	result, err := h.auth.Login(c.Context(), req.Email, req.Password, c.IP())
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			go h.audit.Log(context.Background(), req.Email, "login.failed", "", c.IP())
			return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
		}
		log.Printf("login error: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "login failed")
	}
	go h.audit.Log(context.Background(), result.User.Email, "login.success", "role="+result.User.Role, c.IP())

	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    result.RefreshToken,
		Path:     "/api/auth",
		Expires:  result.RefreshTokenExpiresAt,
		HTTPOnly: true,
		Secure:   h.secureCookies,
		SameSite: fiber.CookieSameSiteStrictMode,
	})

	return c.JSON(fiber.Map{
		"accessToken": result.AccessToken,
		"user": fiber.Map{
			"id":    result.User.ID,
			"email": result.User.Email,
			"role":  result.User.Role,
		},
	})
}

func (h *AuthHandler) refresh(c *fiber.Ctx) error {
	raw := c.Cookies(refreshCookieName)
	if raw == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing refresh token")
	}

	result, err := h.auth.Refresh(c.Context(), raw)
	if err != nil {
		h.clearRefreshCookie(c)
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired refresh token")
	}

	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    result.RefreshToken,
		Path:     "/api/auth",
		Expires:  result.RefreshTokenExpiresAt,
		HTTPOnly: true,
		Secure:   h.secureCookies,
		SameSite: fiber.CookieSameSiteStrictMode,
	})

	return c.JSON(fiber.Map{
		"accessToken": result.AccessToken,
		"user": fiber.Map{
			"id":    result.User.ID,
			"email": result.User.Email,
			"role":  result.User.Role,
		},
	})
}

func (h *AuthHandler) logout(c *fiber.Ctx) error {
	raw := c.Cookies(refreshCookieName)
	if raw != "" {
		_ = h.auth.Logout(c.Context(), raw)
	}
	h.clearRefreshCookie(c)
	return c.JSON(fiber.Map{"ok": true})
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) forgotPassword(c *fiber.Ctx) error {
	var req forgotPasswordRequest
	if err := c.BodyParser(&req); err != nil || req.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}

	if _, _, err := h.auth.ForgotPassword(c.Context(), req.Email); err != nil {
		log.Printf("forgot-password error: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "request failed")
	}

	// Always the same response, whether or not the email exists, so we never
	// leak account existence.
	return c.JSON(fiber.Map{"ok": true})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

func (h *AuthHandler) resetPassword(c *fiber.Ctx) error {
	var req resetPasswordRequest
	if err := c.BodyParser(&req); err != nil || req.Token == "" || len(req.NewPassword) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "token and a password of at least 8 characters are required")
	}

	if err := h.auth.ResetPassword(c.Context(), req.Token, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			return fiber.NewError(fiber.StatusBadRequest, "invalid or expired reset token")
		}
		log.Printf("reset-password error: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "reset failed")
	}

	return c.JSON(fiber.Map{"ok": true})
}

func (h *AuthHandler) me(c *fiber.Ctx) error {
	idStr, _ := c.Locals(middleware.LocalUserID).(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token subject")
	}

	user, err := h.auth.GetUserByID(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}

	return c.JSON(fiber.Map{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	})
}

type updateMeRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewEmail        string `json:"newEmail"`
	NewPassword     string `json:"newPassword"`
}

func (h *AuthHandler) updateMe(c *fiber.Ctx) error {
	idStr, _ := c.Locals(middleware.LocalUserID).(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token subject")
	}

	var req updateMeRequest
	if err := c.BodyParser(&req); err != nil || req.CurrentPassword == "" {
		return fiber.NewError(fiber.StatusBadRequest, "current password is required")
	}
	if req.NewPassword != "" && len(req.NewPassword) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "new password must be at least 8 characters")
	}
	if req.NewEmail == "" && req.NewPassword == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nothing to update")
	}

	user, err := h.auth.UpdateOwnProfile(c.Context(), id, req.CurrentPassword, req.NewEmail, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			return fiber.NewError(fiber.StatusUnauthorized, "current password is incorrect")
		case errors.Is(err, service.ErrEmailTaken):
			return fiber.NewError(fiber.StatusConflict, "that email is already in use")
		default:
			log.Printf("update own profile: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "could not update profile")
		}
	}

	go h.audit.Log(context.Background(), user.Email, "account.self_updated", "", c.IP())
	return c.JSON(fiber.Map{"id": user.ID, "email": user.Email, "role": user.Role})
}

func (h *AuthHandler) logoutOthers(c *fiber.Ctx) error {
	idStr, _ := c.Locals(middleware.LocalUserID).(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token subject")
	}

	raw := c.Cookies(refreshCookieName)
	if raw == "" {
		return fiber.NewError(fiber.StatusBadRequest, "no active session cookie")
	}

	count, err := h.auth.LogoutOtherSessions(c.Context(), id, raw)
	if err != nil {
		log.Printf("logout others: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not log out other sessions")
	}

	user, err := h.auth.GetUserByID(c.Context(), id)
	if err == nil {
		go h.audit.Log(context.Background(), user.Email, "account.logout_others", "", c.IP())
	}
	return c.JSON(fiber.Map{"revoked": count})
}
