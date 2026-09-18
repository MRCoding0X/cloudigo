package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"cloudigo/backend/internal/model"
	"cloudigo/backend/internal/repository"
)

type AdminPagesHandler struct {
	pages *repository.PageRepository
}

func NewAdminPagesHandler(pages *repository.PageRepository) *AdminPagesHandler {
	return &AdminPagesHandler{pages: pages}
}

func (h *AdminPagesHandler) Register(admin fiber.Router) {
	admin.Get("/pages", h.list)
	admin.Post("/pages", h.create)
	admin.Put("/pages/:id", h.update)
	admin.Delete("/pages/:id", h.delete)
}

func (h *AdminPagesHandler) list(c *fiber.Ctx) error {
	pages, err := h.pages.List(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list pages")
	}
	return c.JSON(fiber.Map{"pages": pages})
}

type pageRequest struct {
	Type      string `json:"type"`
	Lang      string `json:"lang"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	SortOrder int    `json:"sortOrder"`
}

func (r pageRequest) toModel() *model.Page {
	pageType := r.Type
	if pageType != "page" && pageType != "terms_page" {
		pageType = "page"
	}
	lang := r.Lang
	if lang == "" {
		lang = "en"
	}
	return &model.Page{Type: pageType, Lang: lang, Title: r.Title, Content: r.Content, SortOrder: r.SortOrder}
}

func (h *AdminPagesHandler) create(c *fiber.Ctx) error {
	var req pageRequest
	if err := c.BodyParser(&req); err != nil || req.Title == "" {
		return fiber.NewError(fiber.StatusBadRequest, "title is required")
	}
	page, err := h.pages.Create(c.Context(), req.toModel())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not create page")
	}
	return c.Status(fiber.StatusCreated).JSON(page)
}

func (h *AdminPagesHandler) update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var req pageRequest
	if err := c.BodyParser(&req); err != nil || req.Title == "" {
		return fiber.NewError(fiber.StatusBadRequest, "title is required")
	}
	page, err := h.pages.Update(c.Context(), id, req.toModel())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "page not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "could not update page")
	}
	return c.JSON(page)
}

func (h *AdminPagesHandler) delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.pages.Delete(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not delete page")
	}
	return c.JSON(fiber.Map{"ok": true})
}
