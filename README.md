# Cloudigo

Cloudigo is a self-hosted file-sharing app - upload a file, get a link, share it.

## What it does

- Drag-and-drop or folder uploads, sent in chunks so large files don't choke a single request, with automatic retry if the connection drops mid-upload
- Share by link or by email to a list of recipients. Link shares get their own safe, download-only code - kept separate from the private link the uploader uses to manage the upload later (edit the password/expiry, delete it early). Nobody who just has the share link can touch either of those.
- Optional password protection, self-destruct after the first download, and a custom expiry
- QR codes for share links, live upload speed/ETA, and a "someone just downloaded this" notice that shows up in real time
- Optional AES-256-GCM encryption at rest - still supports HTTP range requests, so resuming or seeking a download works even on encrypted files
- A full admin panel underneath it all: uploads/downloads/users management with search and CSV export, an audit log, an IP blocklist, editable email templates you can test-send before relying on, a settings page with export/import for backups, and a system-info page that warns when disk space is getting low

## Stack

- **Backend:** Go, Fiber, PostgreSQL (via pgx), golang-migrate
- **Frontend:** Next.js (App Router, TypeScript, Tailwind)
- **Realtime:** WebSocket, for upload/download progress and live admin notifications

## Running it locally

You'll need PostgreSQL, Go 1.27+, and Node 24+.

```bash
# backend
cd backend
cp .env.example .env   # point DATABASE_URL at your local Postgres
go run ./cmd/api        # runs migrations on startup, listens on :8080

# frontend
cd frontend
npm install
npm run dev              # :3000
```

Create the first admin account with:

```bash
go run ./cmd/bootstrapadmin -email you@example.com -password <at least 8 characters>
```

SMTP is configured from the admin panel (Settings → Mail), not environment variables - without a configured host, outgoing mail just gets logged to the console instead of sent, which is fine for local dev.

## Deploying

`docker-compose.yml` builds Postgres, the backend, and the frontend from the Dockerfiles in each directory. A few things worth getting right before it's actually live:

1. **Put a TLS-terminating reverse proxy in front of it** (nginx, Caddy, Traefik, or your cloud's own load balancer) before setting `APP_ENV=production` on the backend. That flag marks the refresh-token cookie `Secure`, which silently breaks login over plain HTTP if there's no HTTPS yet.
2. **Replace `JWT_ACCESS_SECRET` and `JWT_REFRESH_SECRET`** with real random values (`openssl rand -base64 32` works well) - never ship the dev defaults sitting in `docker-compose.yml`.
3. **Set `FRONTEND_URL`** to wherever the frontend is actually reachable from the outside. It's used to build the download links that go out in emails.
4. **Back up the `cloudigo_storage` volume** - every uploaded file, background image, and the audit log live there - along with `cloudigo_pg_data` for the database itself.

Once that's sorted, deploys are just: build new images, `docker compose up -d --build`, migrations run automatically on backend startup.
