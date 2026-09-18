package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cloudigo/backend/internal/repository"
)

// AuditService replaces the legacy droppy_log table, which had no code path
// that ever wrote to it (see PROJECT_DOCUMENTATION.md §19.5) — every call
// here actually persists, both to the database (for the admin UI) and to a
// plain-text audit.log file (for tooling/grep, and as a backup if the DB
// write itself is what's being audited).
type AuditService struct {
	repo *repository.AuditRepository

	mu      sync.Mutex
	logPath string
}

func NewAuditService(repo *repository.AuditRepository, storagePath string) *AuditService {
	return &AuditService{repo: repo, logPath: filepath.Join(storagePath, "audit.log")}
}

// Log records an audit event. Fire-and-forget by design (called via `go`
// from callers) — an audit-logging failure must never break the action being
// audited.
func (s *AuditService) Log(ctx context.Context, actorEmail, eventType, details, ip string) {
	if err := s.repo.Insert(ctx, actorEmail, eventType, details, ip); err != nil {
		log.Printf("audit: db insert failed for %s: %v", eventType, err)
	}
	s.appendToFile(actorEmail, eventType, details, ip)
}

func (s *AuditService) appendToFile(actorEmail, eventType, details, ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("audit: could not open log file: %v", err)
		return
	}
	defer f.Close()

	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n", time.Now().Format(time.RFC3339), eventType, actorEmail, ip, details)
	if _, err := f.WriteString(line); err != nil {
		log.Printf("audit: could not write log file: %v", err)
	}
}
