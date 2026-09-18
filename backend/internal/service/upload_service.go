package service

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"cloudigo/backend/internal/cryptfile"
	"cloudigo/backend/internal/model"
	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/storage"
	"cloudigo/backend/internal/thumbnail"
	"cloudigo/backend/internal/ws"
)

var (
	ErrUploadNotWritable  = errors.New("upload is not accepting files")
	ErrRateLimited        = errors.New("too many uploads from this address, try again later")
	ErrIPBlocked          = errors.New("this address is not allowed to upload")
	ErrBlockedEmail       = errors.New("this email address is not allowed")
	ErrNoRecipients       = errors.New("at least one recipient is required")
	ErrTooManyRecipients  = errors.New("too many recipients")
	ErrEmailNotVerified   = errors.New("email address not verified")
	ErrNoFiles            = errors.New("upload has no files")
	ErrFileTooLarge       = errors.New("file exceeds the maximum allowed size")
	ErrTooManyFiles       = errors.New("too many files in this upload")
	ErrBlockedFileType    = errors.New("this file type is not allowed")
	ErrInvalidFileID      = errors.New("file id must be a valid UUID")
	ErrVerificationFailed = errors.New("invalid or expired verification code")
	ErrReceiverNotFound   = errors.New("receiver not found")
	ErrNotMailShare       = errors.New("this upload was not shared by email")
)

type UploadService struct {
	uploads      *repository.UploadRepository
	files        *repository.FileRepository
	receivers    *repository.ReceiverRepository
	settingsRepo *repository.SettingsRepository
	emailVerify  *repository.EmailVerificationRepository
	blockedIPs   *repository.BlockedIPRepository
	storage      *storage.Local
	hub          *ws.Hub
	email        *EmailService
	frontendURL  string
}

func NewUploadService(
	uploads *repository.UploadRepository,
	files *repository.FileRepository,
	receivers *repository.ReceiverRepository,
	settingsRepo *repository.SettingsRepository,
	emailVerify *repository.EmailVerificationRepository,
	blockedIPs *repository.BlockedIPRepository,
	store *storage.Local,
	hub *ws.Hub,
	email *EmailService,
	frontendURL string,
) *UploadService {
	return &UploadService{
		uploads: uploads, files: files, receivers: receivers,
		settingsRepo: settingsRepo, emailVerify: emailVerify, blockedIPs: blockedIPs, storage: store, hub: hub,
		email: email, frontendURL: frontendURL,
	}
}

// CreateSession starts a new upload: a placeholder DB row (status=processing)
// files can immediately reference, plus a fresh public upload_id and the
// uploader's own secret management code.
func (s *UploadService) CreateSession(ctx context.Context, ip string) (*model.Upload, error) {
	if blocked, err := s.blockedIPs.IsBlocked(ctx, ip); err != nil {
		return nil, err
	} else if blocked {
		return nil, ErrIPBlocked
	}

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < 5; attempt++ {
		uploadID, err := RandomAlnum(settings.UploadIDLength)
		if err != nil {
			return nil, err
		}
		secretCode, err := RandomHex(20)
		if err != nil {
			return nil, err
		}
		shareCode, err := RandomHex(20)
		if err != nil {
			return nil, err
		}

		upload, err := s.uploads.CreateSession(ctx, uploadID, secretCode, shareCode, ip)
		if err == nil {
			return upload, nil
		}
		// Extremely unlikely upload_id collision — retry with a fresh id.
	}
	return nil, fmt.Errorf("could not allocate a unique upload id")
}

type ChunkResult struct {
	ReceivedBytes int64
	TotalBytes    int64
	Complete      bool
}

// ReceiveChunk appends one byte range of one file to on-disk temp storage.
// Chunks for a given file must arrive in order (offset-ascending); the
// implementation detects "file complete" purely by on-disk size reaching
// totalSize, which is only correct for in-order delivery — true out-of-order
// resumable upload (tus-style) is out of scope for this phase.
func (s *UploadService) ReceiveChunk(ctx context.Context, uploadIDStr, fileIDStr, fileName string, rangeStart, rangeEnd, totalSize int64, body []byte) (*ChunkResult, error) {
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		return nil, ErrInvalidFileID
	}

	upload, err := s.uploads.GetByUploadID(ctx, uploadIDStr)
	if err != nil {
		return nil, err
	}
	if upload.Status != model.StatusProcessing {
		return nil, ErrUploadNotWritable
	}

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}

	maxBytes := int64(settings.MaxUploadSizeMB) * 1024 * 1024
	if maxBytes > 0 && totalSize > maxBytes {
		return nil, ErrFileTooLarge
	}
	if ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), ".")); ext != "" && settings.BlockedFileTypes != "" {
		for _, blocked := range strings.Split(settings.BlockedFileTypes, ",") {
			if strings.EqualFold(strings.TrimSpace(blocked), ext) {
				return nil, ErrBlockedFileType
			}
		}
	}
	if rangeStart == 0 {
		existing, err := s.files.ListByUploadID(ctx, upload.ID)
		if err != nil {
			return nil, err
		}
		if settings.MaxFiles > 0 && len(existing) >= settings.MaxFiles {
			return nil, ErrTooManyFiles
		}
	}

	if err := storage.EnsureDir(s.storage.TempDir(uploadIDStr)); err != nil {
		return nil, err
	}

	partPath := s.storage.PartPath(uploadIDStr, fileIDStr)
	f, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	if _, err := f.WriteAt(body, rangeStart); err != nil {
		f.Close()
		return nil, err
	}
	info, err := f.Stat()
	f.Close()
	if err != nil {
		return nil, err
	}

	result := &ChunkResult{ReceivedBytes: rangeEnd + 1, TotalBytes: totalSize}

	s.hub.Broadcast(ws.UploadRoom(uploadIDStr), map[string]any{
		"type":          "chunk.progress",
		"fileId":        fileIDStr,
		"fileName":      fileName,
		"receivedBytes": result.ReceivedBytes,
		"totalBytes":    totalSize,
	})

	if info.Size() >= totalSize {
		finalTemp := s.storage.CompletedTempPath(uploadIDStr, fileIDStr)
		if err := os.Rename(partPath, finalTemp); err != nil {
			return nil, err
		}

		secretCode, err := RandomHex(16)
		if err != nil {
			return nil, err
		}
		if _, err := s.files.Add(ctx, fileID, upload.ID, secretCode, fileName, info.Size()); err != nil {
			return nil, err
		}

		result.Complete = true
		s.hub.Broadcast(ws.UploadRoom(uploadIDStr), map[string]any{
			"type":     "file.complete",
			"fileId":   fileIDStr,
			"fileName": fileName,
			"size":     info.Size(),
		})
	}

	return result, nil
}

type RegisterInput struct {
	EmailFrom     string
	Message       string
	Recipients    []string
	Password      string
	Destruct      bool
	ShareType     string
	ExpireSeconds *int64
	IP            string
}

func (s *UploadService) Register(ctx context.Context, uploadIDStr string, in RegisterInput) (*model.Upload, error) {
	upload, err := s.uploads.GetByUploadID(ctx, uploadIDStr)
	if err != nil {
		return nil, err
	}
	if upload.Status != model.StatusProcessing {
		return nil, ErrUploadNotWritable
	}

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}

	if settings.IPUploadLimit > 0 && in.IP != "" {
		count, err := s.uploads.CountByIPSince(ctx, in.IP, time.Now().Add(-1*time.Hour))
		if err != nil {
			return nil, err
		}
		if count > settings.IPUploadLimit {
			return nil, ErrRateLimited
		}
	}

	if in.EmailFrom != "" && settings.BlockedEmails != "" {
		lower := strings.ToLower(in.EmailFrom)
		for _, blocked := range strings.Split(settings.BlockedEmails, ",") {
			blocked = strings.ToLower(strings.TrimSpace(blocked))
			if blocked != "" && strings.Contains(lower, blocked) {
				return nil, ErrBlockedEmail
			}
		}
	}

	shareType := in.ShareType
	if shareType != model.ShareLink && shareType != model.ShareMail {
		shareType = settings.DefaultShareType
	}

	if shareType == model.ShareMail {
		if len(in.Recipients) == 0 {
			return nil, ErrNoRecipients
		}
		if settings.MaxRecipients > 0 && len(in.Recipients) > settings.MaxRecipients {
			return nil, ErrTooManyRecipients
		}
	}

	if settings.EmailVerify != "false" && in.EmailFrom != "" {
		verifyWindow := 30 * time.Minute
		if settings.EmailVerify == "once" {
			verifyWindow = 100 * 365 * 24 * time.Hour // "ever verified" for once-mode
		}
		verified, err := s.emailVerify.HasVerified(ctx, in.EmailFrom, verifyWindow)
		if err != nil {
			return nil, err
		}
		if !verified {
			return nil, ErrEmailNotVerified
		}
	}

	var passwordHash *string
	if in.Password != "" && settings.PasswordEnabled {
		h, err := HashPassword(in.Password)
		if err != nil {
			return nil, err
		}
		passwordHash = &h
	}

	destruct := in.Destruct && settings.DestructEnabled

	var expiresAt *time.Time
	expireSeconds := settings.DefaultExpireSeconds
	if in.ExpireSeconds != nil {
		expireSeconds = *in.ExpireSeconds
	}
	if expireSeconds > 0 {
		t := time.Now().Add(time.Duration(expireSeconds) * time.Second)
		expiresAt = &t
	}

	err = s.uploads.Register(ctx, upload.ID, repository.RegisterParams{
		EmailFrom: in.EmailFrom, Message: in.Message, PasswordHash: passwordHash,
		Destruct: destruct, ShareType: shareType, ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, err
	}

	if shareType == model.ShareMail {
		for _, email := range in.Recipients {
			email = strings.TrimSpace(email)
			if email == "" {
				continue
			}
			privateID, err := RandomHex(20)
			if err != nil {
				return nil, err
			}
			if _, err := s.receivers.Add(ctx, upload.ID, email, privateID); err != nil {
				return nil, err
			}
		}
	}

	return s.uploads.GetByID(ctx, upload.ID)
}

func (s *UploadService) RequestEmailVerification(ctx context.Context, email string) error {
	code, err := RandomDigits(4)
	if err != nil {
		return err
	}
	if err := s.emailVerify.Create(ctx, email, code); err != nil {
		return err
	}
	go func() {
		if err := s.email.Send(context.Background(), email, "email_verify", "en", map[string]string{"code": code}); err != nil {
			fmt.Printf("send email_verify to %s failed: %v\n", email, err)
		}
	}()
	return nil
}

func (s *UploadService) ConfirmEmailVerification(ctx context.Context, email, code string) error {
	if err := s.emailVerify.ConfirmPending(ctx, email, code, 1*time.Hour); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrVerificationFailed
		}
		return err
	}
	return nil
}

// Complete finalizes an upload once all files have been received: it
// generates image thumbnails, packages multi-file uploads into a single zip,
// optionally encrypts the final artifact (seekable AES-256-GCM, see
// internal/cryptfile), and marks the upload ready.
func (s *UploadService) Complete(ctx context.Context, uploadIDStr string) (*model.Upload, error) {
	upload, err := s.uploads.GetByUploadID(ctx, uploadIDStr)
	if err != nil {
		return nil, err
	}
	if upload.Status != model.StatusProcessing {
		return nil, ErrUploadNotWritable
	}

	files, err := s.files.ListByUploadID(ctx, upload.ID)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, ErrNoFiles
	}

	if err := storage.EnsureDir(s.storage.UploadDir(uploadIDStr)); err != nil {
		return nil, err
	}

	tempPathFor := func(f model.File) string {
		return s.storage.CompletedTempPath(uploadIDStr, f.ID.String())
	}

	if err := storage.EnsureDir(filepath.Dir(s.storage.ThumbPath(uploadIDStr, "x"))); err != nil {
		return nil, err
	}
	for _, f := range files {
		thumbPath := s.storage.ThumbPath(uploadIDStr, f.ID.String())
		if err := thumbnail.Generate(tempPathFor(f), thumbPath); err == nil {
			_ = s.files.MarkThumbnail(ctx, f.ID)
		} else if !errors.Is(err, thumbnail.ErrUnsupported) {
			fmt.Printf("thumbnail generation failed for %s: %v\n", f.FileName, err)
		}
	}

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}

	var encryptKey []byte
	if settings.EncryptFiles {
		encryptKey, err = cryptfile.GenerateKey()
		if err != nil {
			return nil, err
		}
	}

	var finalPath string
	if len(files) > 1 {
		finalPath = s.storage.ZipPath(uploadIDStr)
	} else {
		finalPath = s.storage.FinalFilePath(uploadIDStr, files[0].ID.String(), files[0].FileName)
	}

	if err := writeFinalArtifact(finalPath, files, tempPathFor, encryptKey); err != nil {
		return nil, err
	}

	if encryptKey != nil {
		if err := s.uploads.StoreEncryptKey(ctx, upload.ID, base64.StdEncoding.EncodeToString(encryptKey)); err != nil {
			return nil, err
		}
	}

	var totalSize int64
	for _, f := range files {
		totalSize += f.SizeBytes
	}

	if err := os.RemoveAll(s.storage.TempDir(uploadIDStr)); err != nil {
		fmt.Printf("warning: failed to clean up temp dir for %s: %v\n", uploadIDStr, err)
	}

	if err := s.uploads.MarkReady(ctx, upload.ID, len(files), totalSize); err != nil {
		return nil, err
	}

	s.hub.Broadcast(ws.UploadRoom(uploadIDStr), map[string]any{
		"type":     "upload.ready",
		"uploadId": uploadIDStr,
	})
	s.hub.Broadcast(ws.AdminRoom, map[string]any{
		"type":      "upload.ready",
		"uploadId":  uploadIDStr,
		"fileCount": len(files),
		"size":      totalSize,
	})

	updated, err := s.uploads.GetByID(ctx, upload.ID)
	if err != nil {
		return nil, err
	}
	s.sendCompletionEmails(ctx, updated, files, totalSize)
	return updated, nil
}

// sendCompletionEmails fires the "sender" mail (to the uploader, always) and,
// for share=mail, one "receiver" mail per recipient. Fire-and-forget: a slow
// or misconfigured mail server must never hold up complete()'s response.
func (s *UploadService) sendCompletionEmails(ctx context.Context, upload *model.Upload, files []model.File, totalSize int64) {
	names := fileNameList(files)
	size := formatBytes(totalSize)
	downloadURL := s.buildDownloadURL(upload.UploadID, upload.SecretCode)

	if upload.EmailFrom != "" {
		go func() {
			data := map[string]string{"download_url": downloadURL, "file_names": names, "size": size}
			if err := s.email.Send(context.Background(), upload.EmailFrom, "sender", "en", data); err != nil {
				fmt.Printf("send sender email to %s failed: %v\n", upload.EmailFrom, err)
			}
		}()
	}

	if upload.ShareType == model.ShareMail {
		receivers, err := s.receivers.ListByUploadID(ctx, upload.ID)
		if err != nil {
			fmt.Printf("list receivers for %s failed: %v\n", upload.UploadID, err)
			return
		}
		for _, r := range receivers {
			receiverURL := s.buildDownloadURL(upload.UploadID, r.PrivateID)
			go func(email, url string) {
				data := map[string]string{
					"download_url": url, "file_names": names, "size": size,
					"email_from": upload.EmailFrom, "message": upload.Message,
				}
				if err := s.email.Send(context.Background(), email, "receiver", "en", data); err != nil {
					fmt.Printf("send receiver email to %s failed: %v\n", email, err)
				}
			}(r.Email, receiverURL)
		}
	}
}

func (s *UploadService) buildDownloadURL(uploadID, code string) string {
	return fmt.Sprintf("%s/%s/%s", s.frontendURL, uploadID, code)
}

// ResendReceiverEmail re-sends the "receiver" share notification to one
// recipient of a mail-share upload — for when a receiver lost or never got
// their original email, without making the sender re-upload everything.
func (s *UploadService) ResendReceiverEmail(ctx context.Context, uploadID, receiverID uuid.UUID) error {
	upload, err := s.uploads.GetByID(ctx, uploadID)
	if err != nil {
		return err
	}
	if upload.ShareType != model.ShareMail {
		return ErrNotMailShare
	}

	receivers, err := s.receivers.ListByUploadID(ctx, uploadID)
	if err != nil {
		return err
	}
	var receiver *model.Receiver
	for i := range receivers {
		if receivers[i].ID == receiverID {
			receiver = &receivers[i]
			break
		}
	}
	if receiver == nil {
		return ErrReceiverNotFound
	}

	files, err := s.files.ListByUploadID(ctx, uploadID)
	if err != nil {
		return err
	}

	receiverURL := s.buildDownloadURL(upload.UploadID, receiver.PrivateID)
	data := map[string]string{
		"download_url": receiverURL, "file_names": fileNameList(files), "size": formatBytes(upload.TotalSizeBytes),
		"email_from": upload.EmailFrom, "message": upload.Message,
	}
	return s.email.Send(ctx, receiver.Email, "receiver", "en", data)
}

func fileNameList(files []model.File) string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.FileName
	}
	return strings.Join(names, ", ")
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// writeFinalArtifact writes either a single file or a zip of multiple files
// to finalPath, transparently encrypting the output when key is non-nil.
func writeFinalArtifact(finalPath string, files []model.File, tempPathFor func(model.File) string, key []byte) error {
	out, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if len(files) == 1 && key == nil {
		src, err := os.Open(tempPathFor(files[0]))
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(out, src)
		return err
	}

	if len(files) == 1 {
		src, err := os.Open(tempPathFor(files[0]))
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = cryptfile.Encrypt(src, out, key)
		return err
	}

	// Multiple files: zip them, optionally through the encrypting writer.
	if key == nil {
		return buildZip(out, files, tempPathFor)
	}

	pr, pw := io.Pipe()
	zipErrCh := make(chan error, 1)
	go func() {
		zipErrCh <- buildZip(pw, files, tempPathFor)
		pw.Close()
	}()

	_, encErr := cryptfile.Encrypt(pr, out, key)
	zipErr := <-zipErrCh
	if zipErr != nil {
		return zipErr
	}
	return encErr
}

func buildZip(w io.Writer, files []model.File, tempPathFor func(model.File) string) error {
	zw := zip.NewWriter(w)
	for _, f := range files {
		src, err := os.Open(tempPathFor(f))
		if err != nil {
			return err
		}
		entry, err := zw.Create(f.FileName)
		if err != nil {
			src.Close()
			return err
		}
		if _, err := io.Copy(entry, src); err != nil {
			src.Close()
			return err
		}
		src.Close()
	}
	return zw.Close()
}
