// Package migrations embeds the SQL migration files so the compiled
// binary can run migrations without needing the .sql files on disk.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
