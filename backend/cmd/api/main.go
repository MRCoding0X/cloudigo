package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/robfig/cron/v3"

	"cloudigo/backend/internal/config"
	"cloudigo/backend/internal/db"
	"cloudigo/backend/internal/handler"
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

	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}
	log.Println("migrations: up to date")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("database: connected")

	if err := storage.EnsureDir(cfg.StoragePath); err != nil {
		log.Fatalf("could not prepare storage path %q: %v", cfg.StoragePath, err)
	}
	localStorage := storage.NewLocal(cfg.StoragePath)
	hub := ws.NewHub()

	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(pool)

	settingsRepo := repository.NewSettingsRepository(pool)
	socialRepo := repository.NewSocialRepository(pool)
	pageRepo := repository.NewPageRepository(pool)
	backgroundRepo := repository.NewBackgroundRepository(pool)
	uploadRepo := repository.NewUploadRepository(pool)
	fileRepo := repository.NewFileRepository(pool)
	receiverRepo := repository.NewReceiverRepository(pool)
	downloadRepo := repository.NewDownloadRepository(pool)
	emailVerifyRepo := repository.NewEmailVerificationRepository(pool)
	emailTemplateRepo := repository.NewEmailTemplateRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	blockedIPRepo := repository.NewBlockedIPRepository(pool)

	auditService := service.NewAuditService(auditRepo, cfg.StoragePath)
	emailService := service.NewEmailService(emailTemplateRepo, settingsRepo)
	authService := service.NewAuthService(
		userRepo, refreshTokenRepo, emailService, cfg.FrontendURL,
		cfg.JWTAccessSecret, cfg.JWTRefreshSecret,
		cfg.AccessTokenTTLMin, cfg.RefreshTokenTTLDay,
	)
	uploadService := service.NewUploadService(uploadRepo, fileRepo, receiverRepo, settingsRepo, emailVerifyRepo, blockedIPRepo, localStorage, hub, emailService, cfg.FrontendURL)
	downloadService := service.NewDownloadService(uploadRepo, fileRepo, receiverRepo, downloadRepo, authService, localStorage, hub, emailService)
	cleanupService := service.NewCleanupService(uploadRepo, fileRepo, emailVerifyRepo, localStorage, downloadService)

	scheduler := cron.New()
	if _, err := scheduler.AddFunc(fmt.Sprintf("@every %dm", cfg.CleanupIntervalMin), func() {
		report := cleanupService.RunAll(context.Background())
		log.Printf("cleanup run: expired=%d stuck=%d temp_purged=%d old_verifications=%d",
			report.ExpiredUploadsDeleted, report.StuckUploadsDeleted, report.TempEntriesPurged, report.OldVerificationsPurged)
	}); err != nil {
		log.Fatalf("could not schedule cleanup job: %v", err)
	}
	scheduler.Start()
	defer scheduler.Stop()

	secureCookies := cfg.AppEnv == "production"
	authHandler := handler.NewAuthHandler(authService, auditService, secureCookies)
	uploadHandler := handler.NewUploadHandler(uploadService)
	downloadHandler := handler.NewDownloadHandler(downloadService)
	dashboardHandler := handler.NewAdminDashboardHandler(uploadRepo)
	adminUploadsHandler := handler.NewAdminUploadsHandler(uploadRepo, receiverRepo, downloadService, uploadService, auditService)
	adminDownloadsHandler := handler.NewAdminDownloadsHandler(downloadRepo)
	adminUsersHandler := handler.NewAdminUsersHandler(userRepo, auditService)
	adminPagesHandler := handler.NewAdminPagesHandler(pageRepo)
	adminBackgroundsHandler := handler.NewAdminBackgroundsHandler(backgroundRepo, localStorage)
	adminSettingsHandler := handler.NewAdminSettingsHandler(settingsRepo, socialRepo, auditService)
	adminEmailTemplatesHandler := handler.NewAdminEmailTemplatesHandler(emailTemplateRepo, settingsRepo, emailService, auditService)
	systemService := service.NewSystemService(pool, cfg.StoragePath)
	adminSystemHandler := handler.NewAdminSystemHandler(systemService)
	adminAuditHandler := handler.NewAdminAuditHandler(auditRepo)
	adminBrandingHandler := handler.NewAdminBrandingHandler(settingsRepo, localStorage, auditService)
	adminBlockedIPsHandler := handler.NewAdminBlockedIPsHandler(blockedIPRepo, auditService)
	publicSettingsHandler := handler.NewPublicSettingsHandler(settingsRepo, socialRepo)
	contactHandler := handler.NewContactHandler(settingsRepo, emailService)
	publicPagesHandler := handler.NewPublicPagesHandler(pageRepo)

	app := fiber.New(fiber.Config{
		AppName:   "Cloudigo API",
		BodyLimit: 32 * 1024 * 1024, // generous headroom above the default chunk size
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var fe *fiber.Error
			if errors.As(err, &fe) {
				code = fe.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(recover.New())
	app.Use(helmet.New(helmet.Config{
		// Frontend and backend are separate origins (different ports in dev,
		// commonly still separate in prod), so the default "same-origin"
		// policy silently blocks every cross-origin <img> load — background
		// images, download thumbnails, and the branding logo/favicon.
		CrossOriginResourcePolicy: "cross-origin",
	}))
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000",
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Content-Range, X-File-Name",
		ExposeHeaders:    "Content-Range, Content-Disposition, Content-Length",
	}))

	app.Get("/healthz", func(c *fiber.Ctx) error {
		if err := pool.Ping(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "error",
				"error":  "database unreachable",
			})
		}
		return c.JSON(fiber.Map{
			"status": "ok",
			"app":    "cloudigo-api",
		})
	})

	api := app.Group("/api")
	authHandler.Register(api)
	uploadHandler.Register(api)
	downloadHandler.Register(api)
	publicSettingsHandler.Register(api)
	publicPagesHandler.Register(api)
	contactHandler.Register(api)

	adminGroup := handler.NewAdminGroup(api, authService)
	dashboardHandler.Register(adminGroup)
	adminUploadsHandler.Register(adminGroup)
	adminDownloadsHandler.Register(adminGroup)
	adminUsersHandler.Register(adminGroup)
	adminPagesHandler.Register(adminGroup)
	adminBackgroundsHandler.Register(adminGroup, api)
	adminSettingsHandler.Register(adminGroup)
	adminEmailTemplatesHandler.Register(adminGroup)
	adminSystemHandler.Register(adminGroup)
	adminAuditHandler.Register(adminGroup)
	adminBrandingHandler.Register(adminGroup, api)
	adminBlockedIPsHandler.Register(adminGroup)

	handler.RegisterWebSocket(app, hub, authService)

	log.Printf("listening on :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
