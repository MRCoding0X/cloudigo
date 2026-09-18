package handler

import (
	"encoding/csv"
	"log"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/repository"
)

type AdminDownloadsHandler struct {
	downloads *repository.DownloadRepository
}

func NewAdminDownloadsHandler(downloads *repository.DownloadRepository) *AdminDownloadsHandler {
	return &AdminDownloadsHandler{downloads: downloads}
}

func (h *AdminDownloadsHandler) Register(admin fiber.Router) {
	admin.Get("/downloads", func(c *fiber.Ctx) error {
		offset, limit := paginationParams(c)
		rows, total, err := h.downloads.ListForAdmin(c.Context(), c.Query("search"), offset, limit)
		if err != nil {
			log.Printf("list downloads: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "could not list downloads")
		}

		out := make([]fiber.Map, len(rows))
		for i, r := range rows {
			out[i] = fiber.Map{"uploadId": r.UploadID, "email": r.Email, "ip": r.IP, "downloadedAt": r.DownloadedAt}
		}
		return c.JSON(fiber.Map{"downloads": out, "total": total})
	})

	admin.Get("/downloads/export", h.export)
}

func (h *AdminDownloadsHandler) export(c *fiber.Ctx) error {
	rows, err := h.downloads.ListAllForExport(c.Context(), c.Query("search"))
	if err != nil {
		log.Printf("export downloads: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not export downloads")
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", `attachment; filename="downloads.csv"`)

	w := csv.NewWriter(c.Response().BodyWriter())
	_ = w.Write([]string{"upload_id", "email", "ip", "downloaded_at"})
	for _, r := range rows {
		_ = w.Write([]string{r.UploadID, r.Email, r.IP, r.DownloadedAt})
	}
	w.Flush()
	return w.Error()
}
