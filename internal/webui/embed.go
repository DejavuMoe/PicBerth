// Package webui exposes the compiled SPA (built by Vite into ./dist) to the HTTP server.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

// FS returns the SPA build output rooted at dist/.
func FS() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err) // dist is guaranteed to exist by the embed directive
	}
	return sub
}
