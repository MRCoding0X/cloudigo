package storage

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Local resolves on-disk paths for upload storage. All uploads live under
// StoragePath/temp (in-progress chunk assembly) and StoragePath/uploads
// (finalized files, zips, thumbnails), keyed by the public upload_id.
type Local struct {
	BasePath string
}

func NewLocal(basePath string) *Local {
	return &Local{BasePath: basePath}
}

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// SanitizeFilename strips path separators and unusual characters so a
// client-supplied filename is safe to use as (part of) a disk filename.
func SanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._")
	if name == "" {
		name = "file"
	}
	if len(name) > 150 {
		name = name[:150]
	}
	return name
}

func (l *Local) TempDir(uploadID string) string {
	return filepath.Join(l.BasePath, "temp", uploadID)
}

// PartPath is where in-progress chunk bytes are written.
func (l *Local) PartPath(uploadID, fileID string) string {
	return filepath.Join(l.TempDir(uploadID), fileID+".part")
}

// CompletedTempPath is where a fully-received file waits until complete().
func (l *Local) CompletedTempPath(uploadID, fileID string) string {
	return filepath.Join(l.TempDir(uploadID), fileID)
}

func (l *Local) UploadDir(uploadID string) string {
	return filepath.Join(l.BasePath, "uploads", uploadID)
}

func (l *Local) FinalFilePath(uploadID, fileRecordID, fileName string) string {
	return filepath.Join(l.UploadDir(uploadID), fileRecordID+"__"+SanitizeFilename(fileName))
}

func (l *Local) ThumbPath(uploadID, fileRecordID string) string {
	return filepath.Join(l.UploadDir(uploadID), "thumbs", fileRecordID+".jpg")
}

func (l *Local) ZipPath(uploadID string) string {
	return filepath.Join(l.UploadDir(uploadID), "archive.zip")
}

func (l *Local) BackgroundsDir() string {
	return filepath.Join(l.BasePath, "backgrounds")
}

func (l *Local) BrandingDir() string {
	return filepath.Join(l.BasePath, "branding")
}

// BrandingPath is used for both the logo and favicon — kind ("logo" or
// "favicon") plus the original extension keeps the two apart and lets a
// re-upload simply overwrite the previous file at the same deterministic path.
func (l *Local) BrandingPath(kind, fileName string) string {
	ext := unsafeFilenameChars.ReplaceAllString(filepath.Ext(fileName), "")
	if ext == "" {
		ext = ".bin"
	}
	return filepath.Join(l.BrandingDir(), kind+ext)
}

func (l *Local) BackgroundPath(id, fileName string) string {
	return filepath.Join(l.BackgroundsDir(), id+"__"+SanitizeFilename(fileName))
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// GlobBrandingFile finds the on-disk logo/favicon file for kind regardless
// of its extension (the settings table only stores the served URL, not the
// original filename).
func GlobBrandingFile(l *Local, kind string) ([]string, error) {
	return filepath.Glob(filepath.Join(l.BrandingDir(), kind+".*"))
}

// RemoveOtherBrandingFiles deletes any existing kind.* files other than
// keepPath, so re-uploading a logo with a different extension doesn't leave
// the previous file behind forever.
func RemoveOtherBrandingFiles(l *Local, kind, keepPath string) {
	matches, _ := GlobBrandingFile(l, kind)
	for _, m := range matches {
		if m != keepPath {
			_ = os.Remove(m)
		}
	}
}
