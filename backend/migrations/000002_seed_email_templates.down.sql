DELETE FROM email_templates WHERE lang = 'en' AND type IN ('sender', 'receiver', 'downloaded', 'destroyed', 'email_verify');
