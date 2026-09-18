package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/repository"
)

// PublicPagesHandler exposes read-only page content (about/terms) to
// unauthenticated visitors — used by the upload page's terms-acceptance
// gate and any other public page rendering.
type PublicPagesHandler struct {
	pages *repository.PageRepository
}

func NewPublicPagesHandler(pages *repository.PageRepository) *PublicPagesHandler {
	return &PublicPagesHandler{pages: pages}
}

func (h *PublicPagesHandler) Register(router fiber.Router) {
	router.Get("/pages/:type/:lang", h.get)
}

func (h *PublicPagesHandler) get(c *fiber.Ctx) error {
	pageType := c.Params("type")
	if pageType != "page" && pageType != "terms_page" {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	}
	lang := c.Params("lang")

	page, err := h.pages.GetByTypeAndLang(c.Context(), pageType, lang)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "page not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "could not load page")
	}
	return c.JSON(page)
}
