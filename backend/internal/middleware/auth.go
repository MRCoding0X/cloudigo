package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/service"
)

const (
	LocalUserID = "user_id"
	LocalRole   = "role"
	LocalEmail  = "email"
)

// RequireAuth validates the Bearer access token and stores the user id/role in c.Locals.
func RequireAuth(auth *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ParseAccessToken(tokenStr)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals(LocalUserID, claims.UserID)
		c.Locals(LocalRole, claims.Role)
		c.Locals(LocalEmail, claims.Email)
		return c.Next()
	}
}

// RequireRole must run after RequireAuth. It rejects requests whose role is not in allowed.
func RequireRole(allowed ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(LocalRole).(string)
		for _, a := range allowed {
			if role == a {
				return c.Next()
			}
		}
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
}
