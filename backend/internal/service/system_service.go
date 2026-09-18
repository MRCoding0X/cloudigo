package service

import (
	"context"
	"io/fs"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AppVersion is a static build identifier — Cloudigo has no update-checking
// service (the legacy app's Proxibolt license/update API was dropped
// entirely in the rewrite, see PROJECT_DOCUMENTATION.md), so this is just
// informational rather than compared against anything remote.
const AppVersion = "1.0.0"

type SystemService struct {
	pool        *pgxpool.Pool
	storagePath string
	startedAt   time.Time
}

func NewSystemService(pool *pgxpool.Pool, storagePath string) *SystemService {
	return &SystemService{pool: pool, storagePath: storagePath, startedAt: time.Now()}
}

type SystemStats struct {
	AppVersion    string
	GoVersion     string
	OS            string
	Arch          string
	NumCPU        int
	NumGoroutine  int
	UptimeSeconds int64

	DBConnected bool
	DBVersion   string
	DBLatencyMs int64
	DBError     string

	StorageBytes int64
	StorageFiles int

	DiskFreeBytes  int64
	DiskTotalBytes int64
}

func (s *SystemService) Stats(ctx context.Context) SystemStats {
	stats := SystemStats{
		AppVersion:    AppVersion,
		GoVersion:     runtime.Version(),
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		NumCPU:        runtime.NumCPU(),
		NumGoroutine:  runtime.NumGoroutine(),
		UptimeSeconds: int64(time.Since(s.startedAt).Seconds()),
	}

	start := time.Now()
	var version string
	if err := s.pool.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		stats.DBError = err.Error()
	} else {
		stats.DBConnected = true
		stats.DBVersion = version
		stats.DBLatencyMs = time.Since(start).Milliseconds()
	}

	var totalBytes int64
	var fileCount int
	_ = filepath.WalkDir(s.storagePath, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		totalBytes += info.Size()
		fileCount++
		return nil
	})
	stats.StorageBytes = totalBytes
	stats.StorageFiles = fileCount

	if free, total, err := diskUsage(s.storagePath); err == nil {
		stats.DiskFreeBytes = int64(free)
		stats.DiskTotalBytes = int64(total)
	}

	return stats
}
