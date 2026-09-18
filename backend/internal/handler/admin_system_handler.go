package handler

import (
	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/service"
)

type AdminSystemHandler struct {
	system *service.SystemService
}

func NewAdminSystemHandler(system *service.SystemService) *AdminSystemHandler {
	return &AdminSystemHandler{system: system}
}

func (h *AdminSystemHandler) Register(admin fiber.Router) {
	admin.Get("/system", h.get)
}

func (h *AdminSystemHandler) get(c *fiber.Ctx) error {
	s := h.system.Stats(c.Context())
	return c.JSON(fiber.Map{
		"appVersion":    s.AppVersion,
		"goVersion":     s.GoVersion,
		"os":            s.OS,
		"arch":          s.Arch,
		"numCpu":        s.NumCPU,
		"numGoroutine":  s.NumGoroutine,
		"uptimeSeconds": s.UptimeSeconds,
		"db": fiber.Map{
			"connected": s.DBConnected,
			"version":   s.DBVersion,
			"latencyMs": s.DBLatencyMs,
			"error":     s.DBError,
		},
		"storage": fiber.Map{
			"bytes": s.StorageBytes,
			"files": s.StorageFiles,
		},
		"disk": fiber.Map{
			"freeBytes":  s.DiskFreeBytes,
			"totalBytes": s.DiskTotalBytes,
		},
	})
}
