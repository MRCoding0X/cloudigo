package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/repository"
)

// PublicSettingsHandler exposes only the branding-relevant subset of
// settings — never SMTP credentials or other admin-only fields — to
// unauthenticated visitors, so the public upload/download/login pages can
// reflect the site name and theme colors admins configure.
type PublicSettingsHandler struct {
	settings *repository.SettingsRepository
	social   *repository.SocialRepository
}

func NewPublicSettingsHandler(settings *repository.SettingsRepository, social *repository.SocialRepository) *PublicSettingsHandler {
	return &PublicSettingsHandler{settings: settings, social: social}
}

func (h *PublicSettingsHandler) Register(router fiber.Router) {
	router.Get("/settings/public", func(c *fiber.Ctx) error {
		s, err := h.settings.Get(c.Context())
		if err != nil {
			log.Printf("get public settings: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "could not load settings")
		}
		social, err := h.social.Get(c.Context())
		if err != nil {
			log.Printf("get public social links: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "could not load settings")
		}
		return c.JSON(fiber.Map{
			"siteName":            s.SiteName,
			"themeColor":          s.ThemeColor,
			"themeColorSecondary": s.ThemeColorSecondary,
			"logoPath":            s.LogoPath,
			"faviconPath":         s.FaviconPath,
			"contactEnabled":      s.ContactEnabled,
			"lockPage":            s.LockPage,
			"acceptTerms":         s.AcceptTerms,
			"passwordEnabled":     s.PasswordEnabled,
			"destructEnabled":     s.DestructEnabled,
			"shareEnabled":        s.ShareEnabled,
			"maxUploadSizeMB":     s.MaxUploadSizeMB,
			"maxFiles":            s.MaxFiles,
			"maxRecipients":       s.MaxRecipients,
			"social":              social,
		})
	})
}
