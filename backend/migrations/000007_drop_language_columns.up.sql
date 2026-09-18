-- The site's next-intl locale routing (en/sv) was fully removed — the site
-- is English-only now — so the UI-language selection and per-upload locale
-- columns are dead weight. email_templates.lang stays: it's still read on
-- every send (GetByTypeAndLang) and lets an admin maintain non-English
-- template variants independent of the site's own language.
ALTER TABLE settings DROP COLUMN language;
ALTER TABLE settings DROP COLUMN enabled_languages;
ALTER TABLE uploads DROP COLUMN lang;
