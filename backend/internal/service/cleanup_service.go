package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/storage"
)

// CleanupService replaces the legacy CronLib: periodic housekeeping that used
// to run via a publicly-reachable /cron URL. Here it's invoked by an
// in-process scheduler (see cmd/api) and by the standalone cmd/cleanup CLI —
// never as an unauthenticated HTTP endpoint.
type CleanupService struct {
	uploads     *repository.UploadRepository
	files       *repository.FileRepository
	emailVerify *repository.EmailVerificationRepository
	storage     *storage.Local
	download    *DownloadService
}

func NewCleanupService(
	uploads *repository.UploadRepository,
	files *repository.FileRepository,
	emailVerify *repository.EmailVerificationRepository,
	store *storage.Local,
	download *DownloadService,
) *CleanupService {
	return &CleanupService{uploads: uploads, files: files, emailVerify: emailVerify, storage: store, download: download}
}

type CleanupReport struct {
	ExpiredUploadsDeleted  int
	StuckUploadsDeleted    int
	TempEntriesPurged      int
	OldVerificationsPurged int64
}

func (s *CleanupService) RunAll(ctx context.Context) CleanupReport {
	var report CleanupReport

	report.ExpiredUploadsDeleted = s.expireUploads(ctx)
	report.StuckUploadsDeleted = s.cleanStuckProcessing(ctx)
	report.TempEntriesPurged = s.purgeOldTempEntries(12 * time.Hour)

	purged, err := s.emailVerify.DeleteOldPending(ctx, 1*time.Hour)
	if err != nil {
		fmt.Printf("cleanup: delete old pending verifications failed: %v\n", err)
	}
	report.OldVerificationsPurged = purged

	return report
}

// expireUploads deletes files for uploads past their expires_at that are
// still marked ready/inactive (mirrors legacy CronLib::checkUploads).
func (s *CleanupService) expireUploads(ctx context.Context) int {
	expired, err := s.uploads.GetExpired(ctx)
	if err != nil {
		fmt.Printf("cleanup: list expired uploads failed: %v\n", err)
		return 0
	}
	for i := range expired {
		s.download.Destroy(ctx, &expired[i], expired[i].UploadID)
	}
	return len(expired)
}

// cleanStuckProcessing removes uploads whose browser tab was closed mid-upload
// and never reached complete() (mirrors legacy CronLib::checkFiles, minus its
// misleading placement on the Files model — see PROJECT_DOCUMENTATION.md §19.6).
func (s *CleanupService) cleanStuckProcessing(ctx context.Context) int {
	stuck, err := s.uploads.GetStuckProcessing(ctx, 12*time.Hour)
	if err != nil {
		fmt.Printf("cleanup: list stuck uploads failed: %v\n", err)
		return 0
	}
	for _, u := range stuck {
		if err := os.RemoveAll(s.storage.TempDir(u.UploadID)); err != nil {
			fmt.Printf("cleanup: remove temp dir for %s failed: %v\n", u.UploadID, err)
		}
		if err := s.uploads.Delete(ctx, u.ID); err != nil {
			fmt.Printf("cleanup: delete stuck upload %s failed: %v\n", u.UploadID, err)
		}
	}
	return len(stuck)
}

// purgeOldTempEntries removes any leftover entry under storage/temp older
// than maxAge — a safety net for chunks whose upload row is already gone
// (e.g. deleted directly, or a crash between chunk-write and DB insert).
func (s *CleanupService) purgeOldTempEntries(maxAge time.Duration) int {
	root := filepath.Join(s.storage.BasePath, "temp")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		fmt.Printf("cleanup: read temp dir failed: %v\n", err)
		return 0
	}

	cutoff := time.Now().Add(-maxAge)
	count := 0
	for _, entry := range entries {
		if entry.Name() == ".gitkeep" {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			fmt.Printf("cleanup: remove %s failed: %v\n", path, err)
			continue
		}
		count++
	}
	return count
}
