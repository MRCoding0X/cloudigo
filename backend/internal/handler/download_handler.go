package handler

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/cryptfile"
	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
)

type DownloadHandler struct {
	downloads *service.DownloadService
}

func NewDownloadHandler(downloads *service.DownloadService) *DownloadHandler {
	return &DownloadHandler{downloads: downloads}
}

func (h *DownloadHandler) Register(router fiber.Router) {
	g := router.Group("/download/:uploadId/:code")
	g.Get("/", h.info)
	g.Post("/ticket", h.ticket)
	g.Get("/file", h.file)
	g.Get("/thumb/:fileId", h.thumb)
	g.Delete("/", authRateLimit(10), h.deleteUpload)
	g.Patch("/", authRateLimit(10), h.updateSettings)
}

func (h *DownloadHandler) info(c *fiber.Ctx) error {
	info, err := h.downloads.GetInfo(c.Context(), c.Params("uploadId"), c.Params("code"))
	if err != nil {
		return downloadServiceError(err)
	}

	files := make([]fiber.Map, len(info.Files))
	for i, f := range info.Files {
		files[i] = fiber.Map{
			"id": f.ID, "fileName": f.FileName, "sizeBytes": f.SizeBytes, "hasThumbnail": f.HasThumbnail,
		}
	}

	return c.JSON(fiber.Map{
		"status": info.Status, "shareType": info.ShareType, "fileCount": info.FileCount,
		"totalSizeBytes": info.TotalSizeBytes, "files": files, "requiresPassword": info.RequiresPassword,
		"isOwner": info.IsOwner, "emailFrom": info.EmailFrom, "message": info.Message,
		"expiresAt": info.ExpiresAt, "shareCode": info.ShareCode,
	})
}

type updateSettingsRequest struct {
	Password      *string `json:"password"`
	ExpireSeconds *int64  `json:"expireSeconds"`
}

func (h *DownloadHandler) updateSettings(c *fiber.Ctx) error {
	var req updateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	upload, err := h.downloads.UpdateSettingsBySecret(c.Context(), c.Params("uploadId"), c.Params("code"), req.Password, req.ExpireSeconds)
	if err != nil {
		return downloadServiceError(err)
	}
	return c.JSON(fiber.Map{
		"requiresPassword": upload.PasswordHash != nil,
		"expiresAt":        upload.ExpiresAt,
	})
}

type ticketRequest struct {
	Password string `json:"password"`
}

func (h *DownloadHandler) ticket(c *fiber.Ctx) error {
	var req ticketRequest
	_ = c.BodyParser(&req)

	ticket, err := h.downloads.IssueTicket(c.Context(), c.Params("uploadId"), c.Params("code"), req.Password)
	if err != nil {
		return downloadServiceError(err)
	}
	return c.JSON(fiber.Map{"ticket": ticket})
}

func (h *DownloadHandler) file(c *fiber.Ctx) error {
	uploadID := c.Params("uploadId")
	code := c.Params("code")
	ticket := c.Query("ticket")

	grant, err := h.downloads.Authorize(c.Context(), uploadID, code, "", ticket, c.IP())
	if err != nil {
		return downloadServiceError(err)
	}

	finalPath := h.downloads.ResolveFinalPath(uploadID, grant.Files)

	filename := uploadID + ".zip"
	if len(grant.Files) == 1 {
		filename = grant.Files[0].FileName
	}
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, sanitizeHeaderValue(filename)))

	key, err := service.DecodeEncryptKey(grant.Upload)
	if err != nil {
		log.Printf("decode encrypt key: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not read file")
	}

	// Self-destruct only fires on a full (non-Range) transfer: a Range probe
	// (video scrubbing, a resumed partial download) must not delete a file
	// the client hasn't actually received in full yet.
	isFullRequest := c.Get("Range") == ""
	finalizeDestroy := func() {
		if grant.ShouldDestroy && isFullRequest {
			h.downloads.Destroy(context.Background(), grant.Upload, uploadID)
		}
	}

	if key == nil {
		return streamPlainFile(c, finalPath, filename, finalizeDestroy)
	}

	return streamEncryptedFile(c, finalPath, key, filename, finalizeDestroy)
}

func (h *DownloadHandler) deleteUpload(c *fiber.Ctx) error {
	if err := h.downloads.DeleteBySecret(c.Context(), c.Params("uploadId"), c.Params("code")); err != nil {
		return downloadServiceError(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *DownloadHandler) thumb(c *fiber.Ctx) error {
	uploadID := c.Params("uploadId")
	code := c.Params("code")
	fileID := c.Params("fileId")

	if _, err := h.downloads.VerifyAccess(c.Context(), uploadID, code, "", c.Query("ticket")); err != nil {
		return downloadServiceError(err)
	}

	thumbPath := h.downloads.ThumbPath(uploadID, fileID)
	if _, err := os.Stat(thumbPath); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "no thumbnail available")
	}
	return streamPlainFile(c, thumbPath, fileID+".jpg", nil)
}

// streamPlainFile serves a file from disk with manual Range handling,
// explicitly opening and closing its own os.File handle per request. This
// deliberately avoids fiber/fasthttp's SendFile, which caches file handles
// internally — on Windows that cache holds the file open long enough that a
// self-destructing upload's cleanup (os.RemoveAll right after the response)
// fails silently because the OS won't delete an open file.
func streamPlainFile(c *fiber.Ctx, path, filename string, onComplete func()) error {
	f, err := os.Open(path)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "file not found")
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return fiber.NewError(fiber.StatusInternalServerError, "could not read file")
	}
	total := stat.Size()

	start, end := int64(0), total-1
	status := fiber.StatusOK
	if rangeHeader := c.Get("Range"); rangeHeader != "" {
		s, e, ok := parseRangeHeader(rangeHeader, total)
		if !ok {
			f.Close()
			c.Set("Content-Range", fmt.Sprintf("bytes */%d", total))
			return fiber.NewError(fiber.StatusRequestedRangeNotSatisfiable, "invalid range")
		}
		start, end = s, e
		status = fiber.StatusPartialContent
		c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, total))
	}

	if _, err := f.Seek(start, io.SeekStart); err != nil {
		f.Close()
		return fiber.NewError(fiber.StatusInternalServerError, "could not read file")
	}

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Set("Content-Type", contentType)
	c.Set("Accept-Ranges", "bytes")
	c.Set("Content-Length", strconv.FormatInt(end-start+1, 10))
	c.Status(status)

	remaining := end - start + 1
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		_, err := io.CopyN(w, f, remaining)
		_ = w.Flush()
		f.Close() // close before onComplete: a self-destruct delete right
		// after must not race an still-open file handle (fails silently on Windows).
		if err != nil && err != io.EOF {
			log.Printf("stream file: %v", err)
			return
		}
		if onComplete != nil {
			onComplete()
		}
	})

	return nil
}

var rangeHeaderPattern = regexp.MustCompile(`^bytes=(\d*)-(\d*)$`)

// parseRangeHeader supports "bytes=start-end", "bytes=start-" and
// "bytes=-suffixLength", the forms browsers actually send.
func parseRangeHeader(header string, total int64) (start, end int64, ok bool) {
	m := rangeHeaderPattern.FindStringSubmatch(header)
	if m == nil {
		return 0, 0, false
	}
	startStr, endStr := m[1], m[2]

	switch {
	case startStr == "" && endStr != "":
		suffix, err := strconv.ParseInt(endStr, 10, 64)
		if err != nil || suffix <= 0 {
			return 0, 0, false
		}
		if suffix > total {
			suffix = total
		}
		return total - suffix, total - 1, true
	case startStr != "" && endStr == "":
		s, err := strconv.ParseInt(startStr, 10, 64)
		if err != nil || s >= total {
			return 0, 0, false
		}
		return s, total - 1, true
	case startStr != "" && endStr != "":
		s, err1 := strconv.ParseInt(startStr, 10, 64)
		e, err2 := strconv.ParseInt(endStr, 10, 64)
		if err1 != nil || err2 != nil || s > e || s >= total {
			return 0, 0, false
		}
		if e >= total {
			e = total - 1
		}
		return s, e, true
	default:
		return 0, 0, false
	}
}

// streamEncryptedFile serves an AES-GCM cryptfile with Range support,
// decrypting only the chunks the requested range actually touches. onComplete
// runs after a fully successful write (the caller decides whether that
// should trigger self-destruct).
func streamEncryptedFile(c *fiber.Ctx, path string, key []byte, filename string, onComplete func()) error {
	r, err := cryptfile.OpenReader(path, key)
	if err != nil {
		log.Printf("open encrypted file: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not open file")
	}

	total := r.TotalSize()
	start, end := int64(0), total-1
	status := fiber.StatusOK

	if rangeHeader := c.Get("Range"); rangeHeader != "" {
		s, e, ok := parseRangeHeader(rangeHeader, total)
		if !ok {
			r.Close()
			c.Set("Content-Range", fmt.Sprintf("bytes */%d", total))
			return fiber.NewError(fiber.StatusRequestedRangeNotSatisfiable, "invalid range")
		}
		start, end = s, e
		status = fiber.StatusPartialContent
		c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, total))
	}

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Set("Content-Type", contentType)
	c.Set("Accept-Ranges", "bytes")
	c.Set("Content-Length", strconv.FormatInt(end-start+1, 10))
	c.Status(status)

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		err := r.WriteRange(w, start, end)
		_ = w.Flush()
		r.Close() // close before onComplete, see streamPlainFile for why
		if err != nil {
			log.Printf("stream encrypted file: %v", err)
			return
		}
		if onComplete != nil {
			onComplete()
		}
	})

	return nil
}

var headerValueSanitizer = regexp.MustCompile(`["\r\n]`)

func sanitizeHeaderValue(v string) string {
	return headerValueSanitizer.ReplaceAllString(v, "_")
}

func downloadServiceError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound), errors.Is(err, service.ErrAccessDenied):
		return fiber.NewError(fiber.StatusNotFound, "not found")
	case errors.Is(err, service.ErrNotAvailable):
		return fiber.NewError(fiber.StatusGone, "no longer available")
	case errors.Is(err, service.ErrPasswordRequired):
		return fiber.NewError(fiber.StatusUnauthorized, "password required")
	case errors.Is(err, service.ErrInvalidPassword):
		return fiber.NewError(fiber.StatusUnauthorized, "incorrect password")
	default:
		log.Printf("download service error: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "something went wrong")
	}
}
