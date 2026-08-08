// Package webui embeds the built admin UI so the Go binary can serve it
// directly (single container, single process). Run `npm ci && npm run
// build` in this directory before `go build`/`go vet`/`go test` on the
// main module — see the root Makefile's `build` target, which does this
// for you.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var distFS embed.FS

// DistFS returns the built frontend's static files rooted at dist/ (so
// "index.html" and "assets/..." are top-level within it).
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
