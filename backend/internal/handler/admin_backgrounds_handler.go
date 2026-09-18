package handler

import (
	"errors"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/storage"
)

type AdminBackgroundsHandler struct {
	backgrounds *repository.BackgroundRepository
	storage     *storage.Local
}

func NewAdminBackgroundsHandler(backgrounds *repository.BackgroundRepository, store *storage.Local) *AdminBackgroundsHandler {
	return &AdminBackgroundsHandler{backgrounds: backgrounds, storage: store}
}

// Register wires both the admin CRUD routes (protected) and the public file
// route (backgrounds are shown on the public upload/download/login pages).
func (h *AdminBackgroundsHandler) Register(admin fiber.Router, public fiber.Router) {
	admin.Get("/backgrounds", h.list)
	admin.Post("/backgrounds", h.create)
	admin.Delete("/backgrounds/:id", h.delete)

	public.Get("/backgrounds/:id/file", h.file)
}

func (h *AdminBackgroundsHandler) list(c *fiber.Ctx) error {
	list, err := h.backgrounds.List(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list backgrounds")
	}
	rows := make([]fiber.Map, len(list))
	for i, b := range list {
		rows[i] = fiber.Map{
			"id": b.ID, "url": b.URL, "durationSeconds": b.DurationSeconds,
			"fileUrl": "/api/backgrounds/" + b.ID.String() + "/file",
		}
	}
	return c.JSON(fiber.Map{"backgrounds": rows})
}

func (h *AdminBackgroundsHandler) create(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "a background image file is required")
	}

	clickURL := c.FormValue("url")
	var duration *int
	if v := c.FormValue("durationSeconds"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			duration = &n
		}
	}

	id := uuid.New()
	if err := storage.EnsureDir(h.storage.BackgroundsDir()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not save file")
	}
	destPath := h.storage.BackgroundPath(id.String(), fileHeader.Filename)
	if err := c.SaveFile(fileHeader, destPath); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not save file")
	}

	bg, err := h.backgrounds.Create(c.Context(), id, fileHeader.Filename, clickURL, duration)
	if err != nil {
		_ = os.Remove(destPath)
		return fiber.NewError(fiber.StatusInternalServerError, "could not save background")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": bg.ID, "url": bg.URL, "durationSeconds": bg.DurationSeconds,
		"fileUrl": "/api/backgrounds/" + bg.ID.String() + "/file",
	})
}

func (h *AdminBackgroundsHandler) delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	bg, err := h.backgrounds.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "background not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "could not load background")
	}
	if err := h.backgrounds.Delete(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not delete background")
	}
	_ = os.Remove(h.storage.BackgroundPath(bg.ID.String(), bg.Src))
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AdminBackgroundsHandler) file(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	bg, err := h.backgrounds.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "background not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "could not load background")
	}
	return streamPlainFile(c, h.storage.BackgroundPath(bg.ID.String(), bg.Src), bg.Src, nil)
}
