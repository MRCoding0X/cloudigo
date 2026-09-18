package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"cloudigo/backend/internal/model"
)

type SettingsRepository struct {
	pool *pgxpool.Pool
}

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

const settingsColumns = `
	site_name, site_url, max_upload_size_mb, max_chunk_size_mb, max_files,
	max_recipients, blocked_file_types, blocked_emails, default_expire_seconds,
	upload_id_length, email_verify, password_enabled, destruct_enabled,
	share_enabled, default_sharetype, encrypt_files, ip_upload_limit,
	smtp_host, smtp_port, smtp_username, smtp_password, email_from_name, email_from_address,
	contact_enabled, contact_email, theme_color, theme_color_secondary, logo_path, favicon_path,
	lock_page, accept_terms`

func (r *SettingsRepository) Get(ctx context.Context) (*model.Settings, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+settingsColumns+` FROM settings WHERE id = 1`)

	var s model.Settings
	if err := row.Scan(
		&s.SiteName, &s.SiteURL, &s.MaxUploadSizeMB, &s.MaxChunkSizeMB, &s.MaxFiles,
		&s.MaxRecipients, &s.BlockedFileTypes, &s.BlockedEmails, &s.DefaultExpireSeconds,
		&s.UploadIDLength, &s.EmailVerify, &s.PasswordEnabled, &s.DestructEnabled,
		&s.ShareEnabled, &s.DefaultShareType, &s.EncryptFiles, &s.IPUploadLimit,
		&s.SMTPHost, &s.SMTPPort, &s.SMTPUsername, &s.SMTPPassword, &s.EmailFromName, &s.EmailFromAddress,
		&s.ContactEnabled, &s.ContactEmail, &s.ThemeColor, &s.ThemeColorSecondary, &s.LogoPath, &s.FaviconPath,
		&s.LockPage, &s.AcceptTerms,
	); err != nil {
		return nil, err
	}
	return &s, nil
}

// Update replaces every admin-editable column with the values in s (a
// full-object PUT — the admin UI always submits the complete, pre-populated
// settings form, so there is no need for a partial-patch representation).
func (r *SettingsRepository) Update(ctx context.Context, s *model.Settings) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE settings SET
			site_name = $1, site_url = $2, max_upload_size_mb = $3, max_chunk_size_mb = $4, max_files = $5,
			max_recipients = $6, blocked_file_types = $7, blocked_emails = $8, default_expire_seconds = $9,
			upload_id_length = $10, email_verify = $11, password_enabled = $12, destruct_enabled = $13,
			share_enabled = $14, default_sharetype = $15, encrypt_files = $16, ip_upload_limit = $17,
			smtp_host = $18, smtp_port = $19, smtp_username = $20, smtp_password = $21,
			email_from_name = $22, email_from_address = $23,
			contact_enabled = $24, contact_email = $25, theme_color = $26, theme_color_secondary = $27,
			logo_path = $28, favicon_path = $29,
			lock_page = $30, accept_terms = $31,
			updated_at = now()
		WHERE id = 1`,
		s.SiteName, s.SiteURL, s.MaxUploadSizeMB, s.MaxChunkSizeMB, s.MaxFiles,
		s.MaxRecipients, s.BlockedFileTypes, s.BlockedEmails, s.DefaultExpireSeconds,
		s.UploadIDLength, s.EmailVerify, s.PasswordEnabled, s.DestructEnabled,
		s.ShareEnabled, s.DefaultShareType, s.EncryptFiles, s.IPUploadLimit,
		s.SMTPHost, s.SMTPPort, s.SMTPUsername, s.SMTPPassword,
		s.EmailFromName, s.EmailFromAddress,
		s.ContactEnabled, s.ContactEmail, s.ThemeColor, s.ThemeColorSecondary,
		s.LogoPath, s.FaviconPath,
		s.LockPage, s.AcceptTerms,
	)
	return err
}
