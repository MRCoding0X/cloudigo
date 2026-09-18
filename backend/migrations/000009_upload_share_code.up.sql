-- Separates the uploader's owner secret from the code handed out to
-- recipients. Previously a "link"-type upload had only secret_code, which
-- doubled as both the owner-management key AND the link the uploader was
-- expected to forward to friends — meaning anyone who received a forwarded
-- link could also delete the upload or change its password/expiry, since
-- the download page's owner check is a plain code match. share_code is the
-- new public, download-only code; secret_code remains owner-only.

ALTER TABLE uploads ADD COLUMN share_code VARCHAR(64) NOT NULL DEFAULT '';
