package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	ShareLink = "link"
	ShareMail = "mail"

	StatusProcessing = "processing"
	StatusReady      = "ready"
	StatusInactive   = "inactive"
	StatusDestroyed  = "destroyed"
)

type Upload struct {
	ID             uuid.UUID
	UploadID       string
	SecretCode     string
	ShareCode      string
	UserID         *uuid.UUID
	EmailFrom      string
	Message        string
	PasswordHash   *string
	Destruct       bool
	ShareType      string
	Status         string
	EncryptKey     *string
	FileCount      int
	TotalSizeBytes int64
	IP             string
	CreatedAt      time.Time
	ExpiresAt      *time.Time
}

type File struct {
	ID           uuid.UUID
	UploadID     uuid.UUID
	SecretCode   string
	FileName     string
	OriginalPath string
	SizeBytes    int64
	HasThumbnail bool
	CreatedAt    time.Time
}

type Receiver struct {
	ID        uuid.UUID
	UploadID  uuid.UUID
	Email     string
	PrivateID string
	CreatedAt time.Time
}

type Download struct {
	ID           uuid.UUID
	UploadID     uuid.UUID
	Email        string
	IP           string
	DownloadedAt time.Time
}

type EmailVerification struct {
	ID        uuid.UUID
	Email     string
	Code      string
	Status    string
	CreatedAt time.Time
}

type EmailTemplate struct {
	ID      uuid.UUID `json:"id"`
	Type    string    `json:"type"`
	Lang    string    `json:"lang"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	Enabled bool      `json:"enabled"`
}

type Settings struct {
	SiteName             string
	SiteURL              string
	MaxUploadSizeMB      int
	MaxChunkSizeMB       int
	MaxFiles             int
	MaxRecipients        int
	BlockedFileTypes     string
	BlockedEmails        string
	DefaultExpireSeconds int64
	UploadIDLength       int
	EmailVerify          string
	PasswordEnabled      bool
	DestructEnabled      bool
	ShareEnabled         bool
	DefaultShareType     string
	EncryptFiles         bool
	IPUploadLimit        int

	SMTPHost         string
	SMTPPort         int
	SMTPUsername     string
	SMTPPassword     string
	EmailFromName    string
	EmailFromAddress string

	ContactEnabled bool
	ContactEmail   string

	ThemeColor          string
	ThemeColorSecondary string
	LogoPath            string
	FaviconPath         string

	LockPage    string // "false" | "both" | "upload" | "download"
	AcceptTerms bool
}

type AuditEntry struct {
	ID         uuid.UUID `json:"id"`
	ActorEmail string    `json:"actorEmail"`
	EventType  string    `json:"eventType"`
	Details    string    `json:"details"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"createdAt"`
}

type BlockedIP struct {
	ID        uuid.UUID `json:"id"`
	IP        string    `json:"ip"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}

type SocialLinks struct {
	Facebook  string `json:"facebook"`
	Twitter   string `json:"twitter"`
	Instagram string `json:"instagram"`
	GitHub    string `json:"github"`
}

type Page struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Lang      string    `json:"lang"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	SortOrder int       `json:"sortOrder"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Background struct {
	ID              uuid.UUID
	Src             string
	URL             string
	DurationSeconds *int
	SortOrder       int
	CreatedAt       time.Time
}
