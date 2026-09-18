DELETE FROM email_templates WHERE type = 'password_reset';

ALTER TABLE email_templates DROP CONSTRAINT email_templates_type_check;
ALTER TABLE email_templates ADD CONSTRAINT email_templates_type_check
    CHECK (type IN ('receiver','sender','destroyed','downloaded','email_verify'));
