package handler

import (
	"encoding/csv"
	"log"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/repository"
)

type AdminAuditHandler struct {
	audit *repository.AuditRepository
}

func NewAdminAuditHandler(audit *repository.AuditRepository) *AdminAuditHandler {
	return &AdminAuditHandler{audit: audit}
}

func (h *AdminAuditHandler) Register(admin fiber.Router) {
	admin.Get("/audit", func(c *fiber.Ctx) error {
		offset, limit := paginationParams(c)
		entries, total, err := h.audit.List(c.Context(), c.Query("search"), offset, limit)
		if err != nil {
			log.Printf("list audit log: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "could not list audit log")
		}
		return c.JSON(fiber.Map{"entries": entries, "total": total})
	})

	admin.Get("/audit/export", h.export)
}

func (h *AdminAuditHandler) export(c *fiber.Ctx) error {
	entries, err := h.audit.ListAllForExport(c.Context(), c.Query("search"))
	if err != nil {
		log.Printf("export audit log: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not export audit log")
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", `attachment; filename="audit-log.csv"`)

	w := csv.NewWriter(c.Response().BodyWriter())
	_ = w.Write([]string{"time", "event", "actor", "ip", "details"})
	for _, e := range entries {
		_ = w.Write([]string{
			e.CreatedAt.Format("2006-01-02T15:04:05Z"),
			e.EventType, e.ActorEmail, e.IP, e.Details,
		})
	}
	w.Flush()
	return w.Error()
}
