package handler

import (
	"context"
	"encoding/csv"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
)

type AdminUploadsHandler struct {
	uploads   *repository.UploadRepository
	receivers *repository.ReceiverRepository
	download  *service.DownloadService
	upload    *service.UploadService
	audit     *service.AuditService
}

func NewAdminUploadsHandler(
	uploads *repository.UploadRepository,
	receivers *repository.ReceiverRepository,
	download *service.DownloadService,
	upload *service.UploadService,
	audit *service.AuditService,
) *AdminUploadsHandler {
	return &AdminUploadsHandler{uploads: uploads, receivers: receivers, download: download, upload: upload, audit: audit}
}

func (h *AdminUploadsHandler) Register(admin fiber.Router) {
	admin.Get("/uploads", h.list)
	admin.Post("/uploads/:id/destroy", h.destroy)
	admin.Get("/uploads/export", h.export)
	admin.Get("/uploads/:id/receivers", h.listReceivers)
	admin.Post("/uploads/:id/receivers/:receiverId/resend", h.resendReceiver)
}

func paginationParams(c *fiber.Ctx) (offset, limit int) {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ = strconv.Atoi(c.Query("limit", "30"))
	if limit < 1 || limit > 200 {
		limit = 30
	}
	return (page - 1) * limit, limit
}

func (h *AdminUploadsHandler) list(c *fiber.Ctx) error {
	offset, limit := paginationParams(c)
	filter := repository.AdminUploadFilter{Status: c.Query("status"), Search: c.Query("search")}

	uploads, total, err := h.uploads.ListForAdmin(c.Context(), filter, offset, limit)
	if err != nil {
		log.Printf("list uploads: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not list uploads")
	}

	rows := make([]fiber.Map, len(uploads))
	for i, u := range uploads {
		rows[i] = fiber.Map{
			"id": u.ID, "uploadId": u.UploadID, "emailFrom": u.EmailFrom, "shareType": u.ShareType,
			"status": u.Status, "fileCount": u.FileCount, "totalSizeBytes": u.TotalSizeBytes,
			"ip": u.IP, "createdAt": u.CreatedAt, "expiresAt": u.ExpiresAt, "destruct": u.Destruct,
			"passwordProtected": u.PasswordHash != nil,
		}
	}

	return c.JSON(fiber.Map{"uploads": rows, "total": total})
}

func (h *AdminUploadsHandler) destroy(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	upload, err := h.uploads.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "upload not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "could not load upload")
	}

	h.download.Destroy(context.Background(), upload, upload.UploadID)
	go h.audit.Log(context.Background(), actorEmail(c), "upload.manually_destroyed", "uploadId="+upload.UploadID, c.IP())
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AdminUploadsHandler) listReceivers(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	receivers, err := h.receivers.ListByUploadID(c.Context(), id)
	if err != nil {
		log.Printf("list receivers: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not list receivers")
	}
	rows := make([]fiber.Map, len(receivers))
	for i, r := range receivers {
		rows[i] = fiber.Map{"id": r.ID, "email": r.Email, "createdAt": r.CreatedAt}
	}
	return c.JSON(fiber.Map{"receivers": rows})
}

func (h *AdminUploadsHandler) resendReceiver(c *fiber.Ctx) error {
	uploadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	receiverID, err := uuid.Parse(c.Params("receiverId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid receiver id")
	}

	if err := h.upload.ResendReceiverEmail(c.Context(), uploadID, receiverID); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound), errors.Is(err, service.ErrReceiverNotFound):
			return fiber.NewError(fiber.StatusNotFound, "upload or receiver not found")
		case errors.Is(err, service.ErrNotMailShare):
			return fiber.NewError(fiber.StatusConflict, err.Error())
		default:
			log.Printf("resend receiver email: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "could not resend email")
		}
	}

	go h.audit.Log(context.Background(), actorEmail(c), "upload.receiver_resent", "uploadId="+uploadID.String()+" receiverId="+receiverID.String(), c.IP())
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AdminUploadsHandler) export(c *fiber.Ctx) error {
	filter := repository.AdminUploadFilter{Status: c.Query("status"), Search: c.Query("search")}
	uploads, err := h.uploads.ListAllForExport(c.Context(), filter)
	if err != nil {
		log.Printf("export uploads: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not export uploads")
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", `attachment; filename="uploads.csv"`)

	w := csv.NewWriter(c.Response().BodyWriter())
	_ = w.Write([]string{"upload_id", "email_from", "share_type", "status", "file_count", "total_size_bytes", "ip", "created_at", "expires_at"})
	for _, u := range uploads {
		expires := ""
		if u.ExpiresAt != nil {
			expires = u.ExpiresAt.Format(time.RFC3339)
		}
		_ = w.Write([]string{
			u.UploadID, u.EmailFrom, u.ShareType, u.Status,
			strconv.Itoa(u.FileCount), strconv.FormatInt(u.TotalSizeBytes, 10),
			u.IP, u.CreatedAt.Format(time.RFC3339), expires,
		})
	}
	w.Flush()
	return w.Error()
}
