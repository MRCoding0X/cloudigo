-- Fas 5 follow-up: expose settings columns that existed since Fas 0 but were
-- never actually wired into any handler (lock_page, accept_terms, language),
-- plus a new column so the admin can control which bundled UI languages
-- appear in the language switcher.

ALTER TABLE settings ADD COLUMN enabled_languages VARCHAR(100) NOT NULL DEFAULT 'en,sv';
