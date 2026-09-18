package handler

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/model"
	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
)

type AdminSettingsHandler struct {
	settings *repository.SettingsRepository
	social   *repository.SocialRepository
	audit    *service.AuditService
}

func NewAdminSettingsHandler(settings *repository.SettingsRepository, social *repository.SocialRepository, audit *service.AuditService) *AdminSettingsHandler {
	return &AdminSettingsHandler{settings: settings, social: social, audit: audit}
}

func (h *AdminSettingsHandler) Register(admin fiber.Router) {
	admin.Get("/settings", h.get)
	admin.Put("/settings", h.update)
	admin.Get("/settings/export", h.export)
	admin.Post("/settings/import", h.importSettings)
	admin.Get("/social", h.getSocial)
	admin.Put("/social", h.updateSocial)
}

func settingsJSON(s *model.Settings) fiber.Map {
	return fiber.Map{
		"siteName": s.SiteName, "siteUrl": s.SiteURL,
		"maxUploadSizeMb": s.MaxUploadSizeMB, "maxChunkSizeMb": s.MaxChunkSizeMB, "maxFiles": s.MaxFiles,
		"maxRecipients": s.MaxRecipients, "blockedFileTypes": s.BlockedFileTypes, "blockedEmails": s.BlockedEmails,
		"defaultExpireSeconds": s.DefaultExpireSeconds, "uploadIdLength": s.UploadIDLength,
		"emailVerify": s.EmailVerify, "passwordEnabled": s.PasswordEnabled, "destructEnabled": s.DestructEnabled,
		"shareEnabled": s.ShareEnabled, "defaultShareType": s.DefaultShareType, "encryptFiles": s.EncryptFiles,
		"ipUploadLimit": s.IPUploadLimit,
		"smtpHost":      s.SMTPHost, "smtpPort": s.SMTPPort, "smtpUsername": s.SMTPUsername,
		"smtpPassword": s.SMTPPassword, "emailFromName": s.EmailFromName, "emailFromAddress": s.EmailFromAddress,
		"contactEnabled": s.ContactEnabled, "contactEmail": s.ContactEmail,
		"themeColor": s.ThemeColor, "themeColorSecondary": s.ThemeColorSecondary,
		"logoPath": s.LogoPath, "faviconPath": s.FaviconPath,
		"lockPage": s.LockPage, "acceptTerms": s.AcceptTerms,
	}
}

func (h *AdminSettingsHandler) get(c *fiber.Ctx) error {
	s, err := h.settings.Get(c.Context())
	if err != nil {
		log.Printf("get settings: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not load settings")
	}
	return c.JSON(settingsJSON(s))
}

type settingsRequest struct {
	SiteName             string `json:"siteName"`
	SiteURL              string `json:"siteUrl"`
	MaxUploadSizeMB      int    `json:"maxUploadSizeMb"`
	MaxChunkSizeMB       int    `json:"maxChunkSizeMb"`
	MaxFiles             int    `json:"maxFiles"`
	MaxRecipients        int    `json:"maxRecipients"`
	BlockedFileTypes     string `json:"blockedFileTypes"`
	BlockedEmails        string `json:"blockedEmails"`
	DefaultExpireSeconds int64  `json:"defaultExpireSeconds"`
	UploadIDLength       int    `json:"uploadIdLength"`
	EmailVerify          string `json:"emailVerify"`
	PasswordEnabled      bool   `json:"passwordEnabled"`
	DestructEnabled      bool   `json:"destructEnabled"`
	ShareEnabled         bool   `json:"shareEnabled"`
	DefaultShareType     string `json:"defaultShareType"`
	EncryptFiles         bool   `json:"encryptFiles"`
	IPUploadLimit        int    `json:"ipUploadLimit"`
	SMTPHost             string `json:"smtpHost"`
	SMTPPort             int    `json:"smtpPort"`
	SMTPUsername         string `json:"smtpUsername"`
	SMTPPassword         string `json:"smtpPassword"`
	EmailFromName        string `json:"emailFromName"`
	EmailFromAddress     string `json:"emailFromAddress"`
	ContactEnabled       bool   `json:"contactEnabled"`
	ContactEmail         string `json:"contactEmail"`
	ThemeColor           string `json:"themeColor"`
	ThemeColorSecondary  string `json:"themeColorSecondary"`
	LogoPath             string `json:"logoPath"`
	FaviconPath          string `json:"faviconPath"`
	LockPage             string `json:"lockPage"`
	AcceptTerms          bool   `json:"acceptTerms"`
}

// saveSettings validates req and persists it, sharing logic between the
// regular update endpoint and the JSON-import endpoint.
func (h *AdminSettingsHandler) saveSettings(ctx context.Context, req settingsRequest) (*model.Settings, error) {
	if req.EmailVerify != "false" && req.EmailVerify != "once" && req.EmailVerify != "always" {
		req.EmailVerify = "false"
	}
	if req.DefaultShareType != "link" && req.DefaultShareType != "mail" {
		req.DefaultShareType = "link"
	}
	if req.LockPage != "false" && req.LockPage != "both" && req.LockPage != "upload" && req.LockPage != "download" {
		req.LockPage = "false"
	}

	// A blank password field means "leave unchanged" — the GET response
	// echoes the real password back so the form always has one to submit,
	// but we don't want an admin accidentally blanking it by clearing the
	// field and re-saving unrelated settings.
	current, err := h.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	smtpPassword := req.SMTPPassword
	if smtpPassword == "" {
		smtpPassword = current.SMTPPassword
	}

	s := &model.Settings{
		SiteName: req.SiteName, SiteURL: req.SiteURL,
		MaxUploadSizeMB: req.MaxUploadSizeMB, MaxChunkSizeMB: req.MaxChunkSizeMB, MaxFiles: req.MaxFiles,
		MaxRecipients: req.MaxRecipients, BlockedFileTypes: req.BlockedFileTypes, BlockedEmails: req.BlockedEmails,
		DefaultExpireSeconds: req.DefaultExpireSeconds, UploadIDLength: req.UploadIDLength,
		EmailVerify: req.EmailVerify, PasswordEnabled: req.PasswordEnabled, DestructEnabled: req.DestructEnabled,
		ShareEnabled: req.ShareEnabled, DefaultShareType: req.DefaultShareType, EncryptFiles: req.EncryptFiles,
		IPUploadLimit: req.IPUploadLimit,
		SMTPHost:      req.SMTPHost, SMTPPort: req.SMTPPort, SMTPUsername: req.SMTPUsername,
		SMTPPassword: smtpPassword, EmailFromName: req.EmailFromName, EmailFromAddress: req.EmailFromAddress,
		ContactEnabled: req.ContactEnabled, ContactEmail: req.ContactEmail,
		ThemeColor: req.ThemeColor, ThemeColorSecondary: req.ThemeColorSecondary,
		LogoPath: req.LogoPath, FaviconPath: req.FaviconPath,
		LockPage: req.LockPage, AcceptTerms: req.AcceptTerms,
	}

	if err := h.settings.Update(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (h *AdminSettingsHandler) update(c *fiber.Ctx) error {
	var req settingsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	s, err := h.saveSettings(c.Context(), req)
	if err != nil {
		log.Printf("update settings: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not save settings")
	}
	go h.audit.Log(context.Background(), actorEmail(c), "settings.updated", "", c.IP())
	return c.JSON(settingsJSON(s))
}

// export bundles settings and social links into one JSON document an admin
// can download as a config backup, or move to a fresh instance.
func (h *AdminSettingsHandler) export(c *fiber.Ctx) error {
	s, err := h.settings.Get(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not load settings")
	}
	social, err := h.social.Get(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not load social links")
	}

	c.Set("Content-Disposition", `attachment; filename="cloudigo-settings.json"`)
	return c.JSON(fiber.Map{
		"settings": settingsJSON(s),
		"social":   fiber.Map{"facebook": social.Facebook, "twitter": social.Twitter, "instagram": social.Instagram, "github": social.GitHub},
	})
}

type importRequest struct {
	Settings settingsRequest   `json:"settings"`
	Social   model.SocialLinks `json:"social"`
}

// import restores settings and social links from a JSON document previously
// produced by export — replacing the current configuration outright.
func (h *AdminSettingsHandler) importSettings(c *fiber.Ctx) error {
	var req importRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	s, err := h.saveSettings(c.Context(), req.Settings)
	if err != nil {
		log.Printf("import settings: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not import settings")
	}
	if err := h.social.Update(c.Context(), &req.Social); err != nil {
		log.Printf("import social links: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not import social links")
	}

	go h.audit.Log(context.Background(), actorEmail(c), "settings.imported", "", c.IP())
	return c.JSON(settingsJSON(s))
}

func (h *AdminSettingsHandler) getSocial(c *fiber.Ctx) error {
	s, err := h.social.Get(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not load social links")
	}
	return c.JSON(fiber.Map{"facebook": s.Facebook, "twitter": s.Twitter, "instagram": s.Instagram, "github": s.GitHub})
}

func (h *AdminSettingsHandler) updateSocial(c *fiber.Ctx) error {
	var req model.SocialLinks
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.social.Update(c.Context(), &req); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not save social links")
	}
	return c.JSON(fiber.Map{"ok": true})
}
