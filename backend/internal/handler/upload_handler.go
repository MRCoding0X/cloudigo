package handler

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
)

type UploadHandler struct {
	uploads *service.UploadService
}

func NewUploadHandler(uploads *service.UploadService) *UploadHandler {
	return &UploadHandler{uploads: uploads}
}

func (h *UploadHandler) Register(router fiber.Router) {
	g := router.Group("/upload")
	g.Post("/session", h.createSession)
	g.Post("/:uploadId/files/:fileId/chunk", h.chunk)
	g.Post("/:uploadId/register", h.register)
	g.Post("/:uploadId/complete", h.complete)
	g.Post("/verify-email/request", h.requestEmailVerification)
	g.Post("/verify-email/confirm", h.confirmEmailVerification)
}

func (h *UploadHandler) createSession(c *fiber.Ctx) error {
	upload, err := h.uploads.CreateSession(c.Context(), c.IP())
	if err != nil {
		if errors.Is(err, service.ErrIPBlocked) {
			return uploadServiceError(err)
		}
		log.Printf("create upload session: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not start upload")
	}
	return c.JSON(fiber.Map{"uploadId": upload.UploadID, "secretCode": upload.SecretCode, "shareCode": upload.ShareCode})
}

var contentRangePattern = regexp.MustCompile(`^bytes (\d+)-(\d+)/(\d+)$`)

func parseContentRange(header string) (start, end, total int64, err error) {
	m := contentRangePattern.FindStringSubmatch(header)
	if m == nil {
		return 0, 0, 0, fmt.Errorf("invalid Content-Range header %q", header)
	}
	start, _ = strconv.ParseInt(m[1], 10, 64)
	end, _ = strconv.ParseInt(m[2], 10, 64)
	total, _ = strconv.ParseInt(m[3], 10, 64)
	return start, end, total, nil
}

func (h *UploadHandler) chunk(c *fiber.Ctx) error {
	uploadID := c.Params("uploadId")
	fileID := c.Params("fileId")

	fileName, err := url.QueryUnescape(c.Get("X-File-Name"))
	if err != nil || fileName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "X-File-Name header is required")
	}

	start, end, total, err := parseContentRange(c.Get("Content-Range"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	result, err := h.uploads.ReceiveChunk(c.Context(), uploadID, fileID, fileName, start, end, total, c.Body())
	if err != nil {
		return uploadServiceError(err)
	}

	return c.JSON(fiber.Map{
		"receivedBytes": result.ReceivedBytes,
		"totalBytes":    result.TotalBytes,
		"complete":      result.Complete,
	})
}

type registerRequest struct {
	EmailFrom     string   `json:"emailFrom"`
	Message       string   `json:"message"`
	Recipients    []string `json:"recipients"`
	Password      string   `json:"password"`
	Destruct      bool     `json:"destruct"`
	ShareType     string   `json:"shareType"`
	ExpireSeconds *int64   `json:"expireSeconds"`
}

func (h *UploadHandler) register(c *fiber.Ctx) error {
	uploadID := c.Params("uploadId")

	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	upload, err := h.uploads.Register(c.Context(), uploadID, service.RegisterInput{
		EmailFrom: req.EmailFrom, Message: req.Message, Recipients: req.Recipients,
		Password: req.Password, Destruct: req.Destruct, ShareType: req.ShareType,
		ExpireSeconds: req.ExpireSeconds, IP: c.IP(),
	})
	if err != nil {
		return uploadServiceError(err)
	}

	return c.JSON(fiber.Map{"uploadId": upload.UploadID, "status": upload.Status})
}

func (h *UploadHandler) complete(c *fiber.Ctx) error {
	uploadID := c.Params("uploadId")

	upload, err := h.uploads.Complete(c.Context(), uploadID)
	if err != nil {
		return uploadServiceError(err)
	}

	return c.JSON(fiber.Map{
		"uploadId":       upload.UploadID,
		"secretCode":     upload.SecretCode,
		"status":         upload.Status,
		"fileCount":      upload.FileCount,
		"totalSizeBytes": upload.TotalSizeBytes,
	})
}

type verifyEmailRequestBody struct {
	Email string `json:"email"`
}

func (h *UploadHandler) requestEmailVerification(c *fiber.Ctx) error {
	var req verifyEmailRequestBody
	if err := c.BodyParser(&req); err != nil || req.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}
	if err := h.uploads.RequestEmailVerification(c.Context(), req.Email); err != nil {
		log.Printf("request email verification: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not send verification code")
	}
	return c.JSON(fiber.Map{"ok": true})
}

type confirmEmailRequestBody struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (h *UploadHandler) confirmEmailVerification(c *fiber.Ctx) error {
	var req confirmEmailRequestBody
	if err := c.BodyParser(&req); err != nil || req.Email == "" || req.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and code are required")
	}
	if err := h.uploads.ConfirmEmailVerification(c.Context(), req.Email, req.Code); err != nil {
		return uploadServiceError(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

// uploadServiceError maps known service-layer sentinel errors to the right
// HTTP status instead of always falling back to 500.
func uploadServiceError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, "upload not found")
	case errors.Is(err, service.ErrUploadNotWritable),
		errors.Is(err, service.ErrNoFiles):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, service.ErrIPBlocked):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrRateLimited):
		return fiber.NewError(fiber.StatusTooManyRequests, err.Error())
	case errors.Is(err, service.ErrBlockedEmail),
		errors.Is(err, service.ErrNoRecipients),
		errors.Is(err, service.ErrTooManyRecipients),
		errors.Is(err, service.ErrEmailNotVerified),
		errors.Is(err, service.ErrFileTooLarge),
		errors.Is(err, service.ErrTooManyFiles),
		errors.Is(err, service.ErrBlockedFileType),
		errors.Is(err, service.ErrInvalidFileID),
		errors.Is(err, service.ErrVerificationFailed):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		log.Printf("upload service error: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "something went wrong")
	}
}
