package handler

import (
	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/middleware"
	"cloudigo/backend/internal/service"
)

// NewAdminGroup returns the shared role=admin-protected router group that
// every admin resource handler (dashboard, uploads, downloads, users, pages,
// backgrounds, settings) registers its routes on.
func NewAdminGroup(router fiber.Router, auth *service.AuthService) fiber.Router {
	admin := router.Group("/admin", middleware.RequireAuth(auth), middleware.RequireRole("admin"))
	admin.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	return admin
}

// actorEmail reads the authenticated admin's email, set by RequireAuth from
// the access token, for attributing audit log entries.
func actorEmail(c *fiber.Ctx) string {
	email, _ := c.Locals(middleware.LocalEmail).(string)
	return email
}
