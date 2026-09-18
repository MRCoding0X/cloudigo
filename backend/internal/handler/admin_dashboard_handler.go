package handler

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/repository"
)

type AdminDashboardHandler struct {
	uploads *repository.UploadRepository
}

func NewAdminDashboardHandler(uploads *repository.UploadRepository) *AdminDashboardHandler {
	return &AdminDashboardHandler{uploads: uploads}
}

func (h *AdminDashboardHandler) Register(admin fiber.Router) {
	admin.Get("/dashboard", func(c *fiber.Ctx) error {
		stats, err := h.uploads.GetDashboardStats(c.Context())
		if err != nil {
			log.Printf("dashboard stats: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "could not load dashboard")
		}
		return c.JSON(fiber.Map{
			"totalUploads":      stats.TotalUploads,
			"activeUploads":     stats.ActiveUploads,
			"destroyedUploads":  stats.DestroyedUploads,
			"totalDownloads":    stats.TotalDownloads,
			"totalStorageBytes": stats.TotalStorageBytes,
		})
	})

	admin.Get("/dashboard/timeseries", func(c *fiber.Ctx) error {
		days, _ := strconv.Atoi(c.Query("days", "30"))
		if days < 1 || days > 365 {
			days = 30
		}
		stats, err := h.uploads.GetDailyStats(c.Context(), days)
		if err != nil {
			log.Printf("dashboard timeseries: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "could not load statistics")
		}
		return c.JSON(fiber.Map{"days": stats})
	})
}
