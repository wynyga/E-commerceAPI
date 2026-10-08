// Package migrations embeds the SQL files so the service binary carries its own schema.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
