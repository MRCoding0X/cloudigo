package handler

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/model"
	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
)

var validEmailTemplateTypes = map[string]bool{
	"sender": true, "receiver": true, "destroyed": true,
	"downloaded": true, "email_verify": true, "password_reset": true,
}

type AdminEmailTemplatesHandler struct {
	templates    *repository.EmailTemplateRepository
	settingsRepo *repository.SettingsRepository
	email        *service.EmailService
	audit        *service.AuditService
}

func NewAdminEmailTemplatesHandler(
	templates *repository.EmailTemplateRepository,
	settingsRepo *repository.SettingsRepository,
	email *service.EmailService,
	audit *service.AuditService,
) *AdminEmailTemplatesHandler {
	return &AdminEmailTemplatesHandler{templates: templates, settingsRepo: settingsRepo, email: email, audit: audit}
}

func (h *AdminEmailTemplatesHandler) Register(admin fiber.Router) {
	admin.Get("/email-templates", h.list)
	admin.Put("/email-templates", h.upsert)
	admin.Post("/email-templates/test", h.test)
}

// sampleTemplateData returns realistic placeholder values per template type,
// mirroring the real data maps built in upload_service.go/download_service.go/
// auth_service.go, so a test send exercises the exact same placeholders a
// real email would.
func sampleTemplateData(templateType string) map[string]string {
	switch templateType {
	case "sender", "receiver":
		return map[string]string{
			"download_url": "https://example.com/aB3dE5fG/download",
			"file_names":   "example-file.pdf, photo.jpg",
			"size":         "12.4 MB",
			"email_from":   "sender@example.com",
			"message":      "This is a sample message included with the upload.",
		}
	case "downloaded":
		return map[string]string{"file_names": "example-file.pdf, photo.jpg", "by_email": "recipient@example.com"}
	case "destroyed":
		return map[string]string{"file_names": "example-file.pdf, photo.jpg"}
	case "email_verify":
		return map[string]string{"code": "123456"}
	case "password_reset":
		return map[string]string{"reset_url": "https://example.com/reset-password?token=sample-token"}
	default:
		return map[string]string{}
	}
}

type testEmailTemplateRequest struct {
	Type string `json:"type"`
	Lang string `json:"lang"`
	To   string `json:"to"`
}

func (h *AdminEmailTemplatesHandler) test(c *fiber.Ctx) error {
	var req testEmailTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	req.To = strings.TrimSpace(req.To)
	if !validEmailTemplateTypes[req.Type] {
		return fiber.NewError(fiber.StatusBadRequest, "unknown template type")
	}
	if req.Lang == "" {
		req.Lang = "en"
	}
	if req.To == "" || !strings.Contains(req.To, "@") {
		return fiber.NewError(fiber.StatusBadRequest, "a valid recipient email is required")
	}

	if err := h.email.Send(c.Context(), req.To, req.Type, req.Lang, sampleTemplateData(req.Type)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not send test email: "+err.Error())
	}

	settings, err := h.settingsRepo.Get(c.Context())
	delivered := err == nil && settings.SMTPHost != ""

	go h.audit.Log(context.Background(), actorEmail(c), "email_template.test_sent", "type="+req.Type+" lang="+req.Lang+" to="+req.To, c.IP())
	return c.JSON(fiber.Map{"ok": true, "delivered": delivered})
}

func (h *AdminEmailTemplatesHandler) list(c *fiber.Ctx) error {
	templates, err := h.templates.List(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list email templates")
	}
	return c.JSON(fiber.Map{"templates": templates})
}

type emailTemplateRequest struct {
	Type    string `json:"type"`
	Lang    string `json:"lang"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Enabled bool   `json:"enabled"`
}

func (h *AdminEmailTemplatesHandler) upsert(c *fiber.Ctx) error {
	var req emailTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if !validEmailTemplateTypes[req.Type] {
		return fiber.NewError(fiber.StatusBadRequest, "unknown template type")
	}
	if req.Lang == "" {
		return fiber.NewError(fiber.StatusBadRequest, "lang is required")
	}
	if req.Subject == "" || req.Body == "" {
		return fiber.NewError(fiber.StatusBadRequest, "subject and body are required")
	}

	t := &model.EmailTemplate{Type: req.Type, Lang: req.Lang, Subject: req.Subject, Body: req.Body, Enabled: req.Enabled}
	if err := h.templates.Upsert(c.Context(), t); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not save email template")
	}
	go h.audit.Log(context.Background(), actorEmail(c), "email_template.updated", "type="+req.Type+" lang="+req.Lang, c.IP())
	return c.JSON(fiber.Map{"ok": true})
}
