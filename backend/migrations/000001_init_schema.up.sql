-- Fas 0: grundschema för Droppy-omskrivningen.
-- gen_random_uuid() är inbyggt sedan PostgreSQL 13, ingen extension krävs.

CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                   VARCHAR(255) NOT NULL UNIQUE,
    password_hash           VARCHAR(255) NOT NULL,
    role                    VARCHAR(20)  NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    ip                      VARCHAR(64),
    reset_token             VARCHAR(255),
    reset_token_expires_at  TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

CREATE TABLE settings (
    id                      SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    site_name               VARCHAR(255) NOT NULL DEFAULT 'Cloudigo',
    site_url                VARCHAR(255) NOT NULL DEFAULT '',
    max_upload_size_mb      INTEGER NOT NULL DEFAULT 1024,
    max_chunk_size_mb       INTEGER NOT NULL DEFAULT 1,
    max_files               INTEGER NOT NULL DEFAULT 10,
    max_recipients          INTEGER NOT NULL DEFAULT 10,
    blocked_file_types      TEXT NOT NULL DEFAULT '',
    blocked_emails          TEXT NOT NULL DEFAULT '',
    default_expire_seconds  BIGINT NOT NULL DEFAULT 1209600,
    upload_id_length        INTEGER NOT NULL DEFAULT 8,
    email_verify            VARCHAR(10) NOT NULL DEFAULT 'false' CHECK (email_verify IN ('false','once','always')),
    password_enabled        BOOLEAN NOT NULL DEFAULT true,
    destruct_enabled        BOOLEAN NOT NULL DEFAULT true,
    share_enabled           BOOLEAN NOT NULL DEFAULT true,
    default_sharetype       VARCHAR(10) NOT NULL DEFAULT 'link' CHECK (default_sharetype IN ('link','mail')),
    default_destruct        BOOLEAN NOT NULL DEFAULT false,
    encrypt_files            BOOLEAN NOT NULL DEFAULT false,
    ip_upload_limit          INTEGER NOT NULL DEFAULT 0,
    lock_page                VARCHAR(10) NOT NULL DEFAULT 'false' CHECK (lock_page IN ('false','both','upload','download')),
    accept_terms             BOOLEAN NOT NULL DEFAULT false,
    contact_enabled          BOOLEAN NOT NULL DEFAULT false,
    contact_email            VARCHAR(255) NOT NULL DEFAULT '',
    recaptcha_site_key       VARCHAR(255) NOT NULL DEFAULT '',
    recaptcha_secret_key     VARCHAR(255) NOT NULL DEFAULT '',
    smtp_host                VARCHAR(255) NOT NULL DEFAULT '',
    smtp_port                INTEGER NOT NULL DEFAULT 587,
    smtp_username            VARCHAR(255) NOT NULL DEFAULT '',
    smtp_password            VARCHAR(255) NOT NULL DEFAULT '',
    email_from_name          VARCHAR(255) NOT NULL DEFAULT 'Cloudigo',
    email_from_address       VARCHAR(255) NOT NULL DEFAULT '',
    theme_color              VARCHAR(20) NOT NULL DEFAULT '#4f46e5',
    theme_color_secondary    VARCHAR(20) NOT NULL DEFAULT '#111827',
    logo_path                VARCHAR(255) NOT NULL DEFAULT '',
    favicon_path             VARCHAR(255) NOT NULL DEFAULT '',
    language                 VARCHAR(10) NOT NULL DEFAULT 'en',
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO settings (id) VALUES (1);

CREATE TABLE social_links (
    id          SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    facebook    VARCHAR(255) NOT NULL DEFAULT '',
    twitter     VARCHAR(255) NOT NULL DEFAULT '',
    instagram   VARCHAR(255) NOT NULL DEFAULT '',
    github      VARCHAR(255) NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO social_links (id) VALUES (1);

CREATE TABLE pages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type        VARCHAR(30) NOT NULL DEFAULT 'page' CHECK (type IN ('page','terms_page')),
    lang        VARCHAR(10) NOT NULL DEFAULT 'en',
    title       VARCHAR(255) NOT NULL DEFAULT '',
    content     TEXT NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_pages_type_lang ON pages(type, lang);

CREATE TABLE backgrounds (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    src               VARCHAR(500) NOT NULL,
    url               VARCHAR(500) NOT NULL DEFAULT '',
    duration_seconds  INTEGER,
    sort_order        INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE email_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type        VARCHAR(30) NOT NULL CHECK (type IN ('receiver','sender','destroyed','downloaded','email_verify')),
    lang        VARCHAR(10) NOT NULL DEFAULT 'en',
    subject     VARCHAR(500) NOT NULL DEFAULT '',
    body        TEXT NOT NULL DEFAULT '',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    UNIQUE (type, lang)
);

CREATE TABLE uploads (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id          VARCHAR(32) NOT NULL UNIQUE,
    secret_code        VARCHAR(64) NOT NULL,
    user_id            UUID REFERENCES users(id) ON DELETE SET NULL,
    email_from         VARCHAR(255) NOT NULL DEFAULT '',
    message            TEXT NOT NULL DEFAULT '',
    password_hash      VARCHAR(255),
    destruct           BOOLEAN NOT NULL DEFAULT false,
    share_type         VARCHAR(10) NOT NULL DEFAULT 'link' CHECK (share_type IN ('link','mail')),
    status             VARCHAR(20) NOT NULL DEFAULT 'processing' CHECK (status IN ('processing','ready','inactive','destroyed')),
    lang               VARCHAR(10) NOT NULL DEFAULT 'en',
    file_previews      BOOLEAN NOT NULL DEFAULT false,
    encrypt_key        VARCHAR(255),
    file_count         INTEGER NOT NULL DEFAULT 0,
    total_size_bytes   BIGINT NOT NULL DEFAULT 0,
    ip                 VARCHAR(64) NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at         TIMESTAMPTZ
);
CREATE INDEX idx_uploads_upload_id ON uploads(upload_id);
CREATE INDEX idx_uploads_secret_code ON uploads(secret_code);
CREATE INDEX idx_uploads_status ON uploads(status);
CREATE INDEX idx_uploads_ip_created_at ON uploads(ip, created_at);

CREATE TABLE files (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id      UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    secret_code    VARCHAR(64) NOT NULL,
    file_name      VARCHAR(500) NOT NULL,
    original_path  TEXT NOT NULL DEFAULT '',
    size_bytes     BIGINT NOT NULL DEFAULT 0,
    has_thumbnail  BOOLEAN NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_files_upload_id ON files(upload_id);
CREATE INDEX idx_files_secret_code ON files(secret_code);

CREATE TABLE receivers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id   UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    email       VARCHAR(255) NOT NULL,
    private_id  VARCHAR(64) NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_receivers_upload_id ON receivers(upload_id);
CREATE INDEX idx_receivers_email ON receivers(email);

CREATE TABLE downloads (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id      UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    email          VARCHAR(255) NOT NULL DEFAULT '',
    ip             VARCHAR(64) NOT NULL DEFAULT '',
    downloaded_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_downloads_upload_id ON downloads(upload_id);
CREATE INDEX idx_downloads_email ON downloads(email);

CREATE TABLE email_verifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255) NOT NULL,
    code        VARCHAR(10) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','verified')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_email_verifications_email_code ON email_verifications(email, code);
