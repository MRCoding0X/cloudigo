// Command cleanup runs one pass of the housekeeping job (expired uploads,
// stuck uploads, stale temp files, old email-verification codes) and exits.
// Useful for manual/ops-triggered runs and for testing without waiting on
// the in-process scheduler's interval.
package main

import (
	"context"
	"log"

	"cloudigo/backend/internal/config"
	"cloudigo/backend/internal/db"
	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
	"cloudigo/backend/internal/storage"
	"cloudigo/backend/internal/ws"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is not set (see .env.example)")
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	localStorage := storage.NewLocal(cfg.StoragePath)
	hub := ws.NewHub() // no live listeners in a one-shot CLI run; broadcasts are just dropped

	settingsRepo := repository.NewSettingsRepository(pool)
	uploadRepo := repository.NewUploadRepository(pool)
	fileRepo := repository.NewFileRepository(pool)
	receiverRepo := repository.NewReceiverRepository(pool)
	downloadRepo := repository.NewDownloadRepository(pool)
	emailVerifyRepo := repository.NewEmailVerificationRepository(pool)
	emailTemplateRepo := repository.NewEmailTemplateRepository(pool)

	emailService := service.NewEmailService(emailTemplateRepo, settingsRepo)
	authService := service.NewAuthService(nil, nil, emailService, cfg.FrontendURL, cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.AccessTokenTTLMin, cfg.RefreshTokenTTLDay)
	downloadService := service.NewDownloadService(uploadRepo, fileRepo, receiverRepo, downloadRepo, authService, localStorage, hub, emailService)
	cleanupService := service.NewCleanupService(uploadRepo, fileRepo, emailVerifyRepo, localStorage, downloadService)

	report := cleanupService.RunAll(ctx)
	log.Printf("cleanup done: expired_uploads_deleted=%d stuck_uploads_deleted=%d temp_entries_purged=%d old_verifications_purged=%d",
		report.ExpiredUploadsDeleted, report.StuckUploadsDeleted, report.TempEntriesPurged, report.OldVerificationsPurged)
}
