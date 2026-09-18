ALTER TABLE email_templates DROP CONSTRAINT email_templates_type_check;
ALTER TABLE email_templates ADD CONSTRAINT email_templates_type_check
    CHECK (type IN ('receiver','sender','destroyed','downloaded','email_verify','password_reset'));

INSERT INTO email_templates (type, lang, subject, body) VALUES
('password_reset', 'en', 'Reset your password',
 E'Hi,\n\nA password reset was requested for your {site_name} account. Use the link below to set a new password:\n\n{reset_url}\n\nThis link expires in 1 hour. If you did not request this, you can safely ignore this email.')
ON CONFLICT (type, lang) DO NOTHING;
