package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
	"cloudigo/backend/internal/storage"
)

// AdminBrandingHandler handles logo/favicon upload — the one part of
// Settings > Utseende that needs real file storage rather than a plain
// settings field, so it gets its own small handler alongside
// AdminSettingsHandler.
type AdminBrandingHandler struct {
	settings *repository.SettingsRepository
	storage  *storage.Local
	audit    *service.AuditService
}

func NewAdminBrandingHandler(settings *repository.SettingsRepository, store *storage.Local, audit *service.AuditService) *AdminBrandingHandler {
	return &AdminBrandingHandler{settings: settings, storage: store, audit: audit}
}

func (h *AdminBrandingHandler) Register(admin fiber.Router, public fiber.Router) {
	admin.Post("/branding/logo", h.upload("logo"))
	admin.Post("/branding/favicon", h.upload("favicon"))

	public.Get("/branding/:kind/file", h.file)
}

func (h *AdminBrandingHandler) upload(kind string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "an image file is required")
		}

		if err := storage.EnsureDir(h.storage.BrandingDir()); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not save file")
		}
		destPath := h.storage.BrandingPath(kind, fileHeader.Filename)
		if err := c.SaveFile(fileHeader, destPath); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not save file")
		}
		storage.RemoveOtherBrandingFiles(h.storage, kind, destPath)

		current, err := h.settings.Get(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not load settings")
		}
		servedURL := "/api/branding/" + kind + "/file"
		if kind == "logo" {
			current.LogoPath = servedURL
		} else {
			current.FaviconPath = servedURL
		}
		if err := h.settings.Update(c.Context(), current); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not save settings")
		}

		go h.audit.Log(context.Background(), actorEmail(c), "branding."+kind+"_updated", "", c.IP())
		return c.JSON(fiber.Map{"path": servedURL})
	}
}

func (h *AdminBrandingHandler) file(c *fiber.Ctx) error {
	kind := c.Params("kind")
	if kind != "logo" && kind != "favicon" {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	}

	settings, err := h.settings.Get(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not load settings")
	}
	path := settings.LogoPath
	if kind == "favicon" {
		path = settings.FaviconPath
	}
	if path == "" {
		return fiber.NewError(fiber.StatusNotFound, "not set")
	}

	// path is the served URL we generated ("/api/branding/logo/file"); the
	// actual on-disk file always lives at the deterministic BrandingPath for
	// whatever extension was last uploaded, found by globbing since we don't
	// store the original extension separately.
	matches, _ := storage.GlobBrandingFile(h.storage, kind)
	if len(matches) == 0 {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	}
	return streamPlainFile(c, matches[0], matches[0], nil)
}
