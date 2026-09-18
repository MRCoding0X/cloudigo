package handler

import (
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
)

// ContactHandler forwards the public contact form to the site's configured
// contact address, with the sender's own email set as Reply-To — mirroring
// the legacy app's contact() endpoint (see PROJECT_DOCUMENTATION.md §"contact()").
type ContactHandler struct {
	settings *repository.SettingsRepository
	email    *service.EmailService
}

func NewContactHandler(settings *repository.SettingsRepository, email *service.EmailService) *ContactHandler {
	return &ContactHandler{settings: settings, email: email}
}

func (h *ContactHandler) Register(router fiber.Router) {
	router.Post("/contact", authRateLimit(5), h.submit)
}

type contactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

func (h *ContactHandler) submit(c *fiber.Ctx) error {
	var req contactRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Message = strings.TrimSpace(req.Message)
	if req.Name == "" || req.Email == "" || req.Message == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name, email and message are required")
	}
	if !strings.Contains(req.Email, "@") {
		return fiber.NewError(fiber.StatusBadRequest, "a valid email is required")
	}

	settings, err := h.settings.Get(c.Context())
	if err != nil {
		log.Printf("contact: load settings: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "request failed")
	}
	if !settings.ContactEnabled || settings.ContactEmail == "" {
		return fiber.NewError(fiber.StatusNotFound, "contact form is not enabled")
	}

	subject := fmt.Sprintf("[%s] Contact form: %s", settings.SiteName, req.Name)
	body := fmt.Sprintf("From: %s <%s>\n\n%s", req.Name, req.Email, req.Message)

	if err := h.email.SendDirect(c.Context(), settings.ContactEmail, req.Email, subject, body); err != nil {
		log.Printf("contact: send email: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not send message")
	}

	return c.JSON(fiber.Map{"ok": true})
}
