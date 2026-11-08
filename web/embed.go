package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// GetFS returns the embedded filesystem rooted at the built dist assets.
func GetFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
