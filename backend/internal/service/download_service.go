package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"cloudigo/backend/internal/model"
	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/storage"
	"cloudigo/backend/internal/ws"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAccessDenied     = errors.New("access denied")
	ErrNotAvailable     = errors.New("this upload is not available")
	ErrPasswordRequired = errors.New("password required")
	ErrInvalidPassword  = errors.New("incorrect password")
)

type DownloadService struct {
	uploads   *repository.UploadRepository
	files     *repository.FileRepository
	receivers *repository.ReceiverRepository
	downloads *repository.DownloadRepository
	auth      *AuthService
	storage   *storage.Local
	hub       *ws.Hub
	email     *EmailService
}

func NewDownloadService(
	uploads *repository.UploadRepository,
	files *repository.FileRepository,
	receivers *repository.ReceiverRepository,
	downloads *repository.DownloadRepository,
	auth *AuthService,
	store *storage.Local,
	hub *ws.Hub,
	email *EmailService,
) *DownloadService {
	return &DownloadService{uploads: uploads, files: files, receivers: receivers, downloads: downloads, auth: auth, storage: store, hub: hub, email: email}
}

type DownloadInfo struct {
	Status           string
	ShareType        string
	FileCount        int
	TotalSizeBytes   int64
	Files            []model.File
	RequiresPassword bool
	IsOwner          bool
	EmailFrom        string
	Message          string
	ExpiresAt        *time.Time
	// ShareCode is only populated for the owner (empty otherwise) — it lets
	// the owner get back the safe, download-only link to hand out even after
	// leaving the initial post-upload result page.
	ShareCode string
}

// resolveAccess checks that code is either the upload's own secret_code
// (owner — never handed to recipients), its share_code (the public,
// download-only code recipients actually receive for share=link), or, for
// share=mail, a receiver's private_id. It has no side effects — safe to call
// on every page load/poll.
func (s *DownloadService) resolveAccess(ctx context.Context, upload *model.Upload, code string) (isOwner bool, receiverEmail string, err error) {
	if code == upload.SecretCode {
		return true, "", nil
	}
	if upload.ShareCode != "" && code == upload.ShareCode {
		return false, "", nil
	}
	if upload.ShareType == model.ShareMail {
		receiver, err := s.receivers.GetByUploadAndPrivateID(ctx, upload.ID, code)
		if err == nil {
			return false, receiver.Email, nil
		}
	}
	return false, "", ErrAccessDenied
}

func (s *DownloadService) GetInfo(ctx context.Context, uploadIDStr, code string) (*DownloadInfo, error) {
	upload, err := s.uploads.GetByUploadID(ctx, uploadIDStr)
	if err != nil {
		return nil, err
	}
	// A destroyed/inactive upload's file rows are never cleaned up (only the
	// disk files and status change), so without this check a fresh visit to
	// an already-destroyed link would render as if the files were still
	// there — Authorize/VerifyAccess already guard the same way.
	if upload.Status != model.StatusReady {
		return nil, ErrNotAvailable
	}

	isOwner, _, err := s.resolveAccess(ctx, upload, code)
	if err != nil {
		return nil, err
	}

	files, err := s.files.ListByUploadID(ctx, upload.ID)
	if err != nil {
		return nil, err
	}

	shareCode := ""
	if isOwner {
		shareCode = upload.ShareCode
	}

	return &DownloadInfo{
		Status:           upload.Status,
		ShareType:        upload.ShareType,
		FileCount:        len(files),
		TotalSizeBytes:   upload.TotalSizeBytes,
		Files:            files,
		RequiresPassword: upload.PasswordHash != nil,
		IsOwner:          isOwner,
		EmailFrom:        upload.EmailFrom,
		Message:          upload.Message,
		ExpiresAt:        upload.ExpiresAt,
		ShareCode:        shareCode,
	}, nil
}

func (s *DownloadService) IssueTicket(ctx context.Context, uploadIDStr, code, password string) (string, error) {
	upload, err := s.uploads.GetByUploadID(ctx, uploadIDStr)
	if err != nil {
		return "", err
	}
	if _, _, err := s.resolveAccess(ctx, upload, code); err != nil {
		return "", err
	}
	if upload.PasswordHash == nil {
		return s.auth.IssueDownloadTicket(uploadIDStr, code)
	}
	if password == "" {
		return "", ErrPasswordRequired
	}
	if bcrypt.CompareHashAndPassword([]byte(*upload.PasswordHash), []byte(password)) != nil {
		return "", ErrInvalidPassword
	}
	return s.auth.IssueDownloadTicket(uploadIDStr, code)
}

func (s *DownloadService) checkPasswordGate(upload *model.Upload, uploadIDStr, code, password, ticket string) error {
	if upload.PasswordHash == nil {
		return nil
	}
	if ticket != "" {
		claims, err := s.auth.ParseDownloadTicket(ticket)
		if err == nil && claims.UploadID == uploadIDStr && claims.Code == code {
			return nil
		}
	}
	if password != "" {
		if bcrypt.CompareHashAndPassword([]byte(*upload.PasswordHash), []byte(password)) == nil {
			return nil
		}
		return ErrInvalidPassword
	}
	return ErrPasswordRequired
}

// VerifyAccess checks owner/receiver + password/ticket validity without any
// side effects (no download log, no destruct, no broadcast) — used to gate
// thumbnail previews, which shouldn't count as a "real" download.
func (s *DownloadService) VerifyAccess(ctx context.Context, uploadIDStr, code, password, ticket string) (*model.Upload, error) {
	upload, err := s.uploads.GetByUploadID(ctx, uploadIDStr)
	if err != nil {
		return nil, err
	}
	if upload.Status != model.StatusReady {
		return nil, ErrNotAvailable
	}
	if _, _, err := s.resolveAccess(ctx, upload, code); err != nil {
		return nil, err
	}
	if err := s.checkPasswordGate(upload, uploadIDStr, code, password, ticket); err != nil {
		return nil, err
	}
	return upload, nil
}

type AccessGrant struct {
	Upload        *model.Upload
	Files         []model.File
	ShouldDestroy bool
}

// Authorize performs the full, side-effecting access check for an actual
// file download: password/ticket verification, download logging, the live
// "someone downloaded this" broadcast, and self-destruct handling.
func (s *DownloadService) Authorize(ctx context.Context, uploadIDStr, code, password, ticket, ip string) (*AccessGrant, error) {
	upload, err := s.uploads.GetByUploadID(ctx, uploadIDStr)
	if err != nil {
		return nil, err
	}
	if upload.Status != model.StatusReady {
		return nil, ErrNotAvailable
	}

	isOwner, receiverEmail, err := s.resolveAccess(ctx, upload, code)
	if err != nil {
		return nil, err
	}

	if err := s.checkPasswordGate(upload, uploadIDStr, code, password, ticket); err != nil {
		return nil, err
	}

	if err := s.downloads.Insert(ctx, upload.ID, receiverEmail, ip); err != nil {
		return nil, err
	}

	files, err := s.files.ListByUploadID(ctx, upload.ID)
	if err != nil {
		return nil, err
	}

	notify := upload.ShareType == model.ShareLink || (upload.ShareType == model.ShareMail && !isOwner)
	if notify {
		s.hub.Broadcast(ws.UploadRoom(uploadIDStr), map[string]any{
			"type":         "download.happened",
			"email":        receiverEmail,
			"downloadedAt": time.Now().Format(time.RFC3339),
		})
		s.hub.Broadcast(ws.AdminRoom, map[string]any{
			"type":         "download.happened",
			"uploadId":     uploadIDStr,
			"email":        receiverEmail,
			"downloadedAt": time.Now().Format(time.RFC3339),
		})
		if upload.EmailFrom != "" {
			names := fileNameList(files)
			byEmail := ""
			if receiverEmail != "" {
				byEmail = " by " + receiverEmail
			}
			go func() {
				data := map[string]string{"file_names": names, "by_email": byEmail}
				if err := s.email.Send(context.Background(), upload.EmailFrom, "downloaded", "en", data); err != nil {
					fmt.Printf("send downloaded email to %s failed: %v\n", upload.EmailFrom, err)
				}
			}()
		}
	}

	shouldDestroy := false
	if upload.Destruct {
		if upload.ShareType == model.ShareLink {
			shouldDestroy = true
		} else {
			allDone, err := s.downloads.HasAllReceiversDownloaded(ctx, upload.ID)
			if err == nil && allDone {
				shouldDestroy = true
			}
		}
	}

	return &AccessGrant{Upload: upload, Files: files, ShouldDestroy: shouldDestroy}, nil
}

// DeleteBySecret lets an uploader delete their own upload early, using the
// same secret_code that grants them owner access on the download page —
// mirrors the legacy app's self-service "delete my upload" link. Only the
// owner's own code works here (never a receiver's private_id), and an
// already-destroyed upload is treated as not-found rather than erroring.
func (s *DownloadService) DeleteBySecret(ctx context.Context, uploadIDStr, code string) error {
	upload, err := s.uploads.GetByUploadID(ctx, uploadIDStr)
	if err != nil {
		return err
	}
	if code != upload.SecretCode {
		return ErrAccessDenied
	}
	if upload.Status == model.StatusDestroyed {
		return repository.ErrNotFound
	}

	s.Destroy(ctx, upload, uploadIDStr)
	return nil
}

// UpdateSettingsBySecret lets an uploader change their own upload's password
// and/or expiry after the fact, using the same owner secret_code as
// DeleteBySecret. password: nil leaves it unchanged, "" clears it, anything
// else sets/replaces it. expireSeconds: nil leaves expiry unchanged, 0 means
// never expire, >0 sets a new expiry that many seconds from now.
func (s *DownloadService) UpdateSettingsBySecret(ctx context.Context, uploadIDStr, code string, password *string, expireSeconds *int64) (*model.Upload, error) {
	upload, err := s.uploads.GetByUploadID(ctx, uploadIDStr)
	if err != nil {
		return nil, err
	}
	if code != upload.SecretCode {
		return nil, ErrAccessDenied
	}
	if upload.Status != model.StatusReady {
		return nil, ErrNotAvailable
	}

	passwordHash := upload.PasswordHash
	if password != nil {
		if *password == "" {
			passwordHash = nil
		} else {
			h, err := HashPassword(*password)
			if err != nil {
				return nil, err
			}
			passwordHash = &h
		}
	}

	expiresAt := upload.ExpiresAt
	if expireSeconds != nil {
		if *expireSeconds > 0 {
			t := time.Now().Add(time.Duration(*expireSeconds) * time.Second)
			expiresAt = &t
		} else {
			expiresAt = nil
		}
	}

	if err := s.uploads.UpdateOwnerSettings(ctx, upload.ID, passwordHash, expiresAt); err != nil {
		return nil, err
	}
	upload.PasswordHash = passwordHash
	upload.ExpiresAt = expiresAt
	return upload, nil
}

// Destroy physically deletes an upload's files and marks it destroyed. The
// caller (the download handler) must only call this AFTER the file bytes
// have actually been sent to the client — Authorize only decides whether
// destruction should happen, it never performs it, so a self-destructing
// download is never deleted out from under the response that was supposed to
// serve it.
func (s *DownloadService) Destroy(ctx context.Context, upload *model.Upload, uploadIDStr string) {
	if err := os.RemoveAll(s.storage.UploadDir(uploadIDStr)); err != nil {
		// Non-fatal: the DB status change below still takes the upload out of
		// circulation even if a stray file is left on disk.
	}
	_ = s.uploads.UpdateStatus(ctx, upload.ID, model.StatusDestroyed)
	s.hub.Broadcast(ws.UploadRoom(uploadIDStr), map[string]any{
		"type":     "upload.destroyed",
		"uploadId": uploadIDStr,
	})
	s.hub.Broadcast(ws.AdminRoom, map[string]any{
		"type":     "upload.destroyed",
		"uploadId": uploadIDStr,
	})

	if upload.EmailFrom != "" {
		files, err := s.files.ListByUploadID(ctx, upload.ID)
		if err == nil {
			names := fileNameList(files)
			go func() {
				data := map[string]string{"file_names": names}
				if err := s.email.Send(context.Background(), upload.EmailFrom, "destroyed", "en", data); err != nil {
					fmt.Printf("send destroyed email to %s failed: %v\n", upload.EmailFrom, err)
				}
			}()
		}
	}
}

// ResolveFinalPath returns the on-disk path written by UploadService.Complete
// for this upload (a single final file, or the packaged zip).
func (s *DownloadService) ResolveFinalPath(uploadIDStr string, files []model.File) string {
	if len(files) > 1 {
		return s.storage.ZipPath(uploadIDStr)
	}
	return s.storage.FinalFilePath(uploadIDStr, files[0].ID.String(), files[0].FileName)
}

func (s *DownloadService) ThumbPath(uploadIDStr, fileID string) string {
	return s.storage.ThumbPath(uploadIDStr, fileID)
}

func DecodeEncryptKey(upload *model.Upload) ([]byte, error) {
	if upload.EncryptKey == nil {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(*upload.EncryptKey)
}
