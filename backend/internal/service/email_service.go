package service

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"cloudigo/backend/internal/model"
	"cloudigo/backend/internal/repository"
)

// EmailService reads SMTP configuration from the settings table (DB), not
// static env config — this is the same architectural choice the legacy app
// made (see PROJECT_DOCUMENTATION.md §4): admin-editable runtime config that
// takes effect immediately, no redeploy. Env vars only seed the initial
// migration default.
type EmailService struct {
	templates    *repository.EmailTemplateRepository
	settingsRepo *repository.SettingsRepository
}

func NewEmailService(templates *repository.EmailTemplateRepository, settingsRepo *repository.SettingsRepository) *EmailService {
	return &EmailService{templates: templates, settingsRepo: settingsRepo}
}

// Send renders the named template (falling back to English) with data and
// emails it to `to`. Errors are returned for the caller to log — callers
// trigger this fire-and-forget (in a goroutine) so a slow/broken mail server
// never blocks an HTTP response.
func (s *EmailService) Send(ctx context.Context, to, templateType, lang string, data map[string]string) error {
	if to == "" {
		return nil
	}

	tmpl, err := s.templates.GetByTypeAndLang(ctx, templateType, lang)
	if err != nil {
		return fmt.Errorf("load %s template: %w", templateType, err)
	}
	if !tmpl.Enabled {
		return nil
	}

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	data["site_name"] = settings.SiteName

	subject := renderPlaceholders(tmpl.Subject, data)
	body := renderPlaceholders(tmpl.Body, data)

	return s.sendRaw(settings, to, "", subject, body)
}

// SendDirect emails a subject/body as-is, with no template lookup — for
// content that is composed at request time rather than admin-editable, such
// as the contact form forwarding a visitor's message to the site's contact
// address with the visitor's own email set as Reply-To.
func (s *EmailService) SendDirect(ctx context.Context, to, replyTo, subject, body string) error {
	if to == "" {
		return nil
	}
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	return s.sendRaw(settings, to, replyTo, subject, body)
}

func renderPlaceholders(text string, data map[string]string) string {
	pairs := make([]string, 0, len(data)*2)
	for k, v := range data {
		pairs = append(pairs, "{"+k+"}", v)
	}
	return strings.NewReplacer(pairs...).Replace(text)
}

func (s *EmailService) sendRaw(settings *model.Settings, to, replyTo, subject, body string) error {
	from := settings.EmailFromAddress
	if from == "" {
		from = "no-reply@localhost"
	}

	if settings.SMTPHost == "" {
		log.Printf("email (no SMTP host configured, not sent) to=%s subject=%q\n%s", to, subject, body)
		return nil
	}

	addr := fmt.Sprintf("%s:%d", settings.SMTPHost, settings.SMTPPort)

	var auth smtp.Auth
	if settings.SMTPUsername != "" {
		auth = smtp.PlainAuth("", settings.SMTPUsername, settings.SMTPPassword, settings.SMTPHost)
	}

	fromHeader := from
	if settings.EmailFromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", settings.EmailFromName, from)
	}

	msg := buildMIMEMessage(fromHeader, to, replyTo, subject, body)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func buildMIMEMessage(from, to, replyTo, subject, body string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	if replyTo != "" {
		fmt.Fprintf(&b, "Reply-To: %s\r\n", replyTo)
	}
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}
