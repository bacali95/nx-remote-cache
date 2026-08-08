package adminapi

import (
	"io/fs"
	"net/http"
	"strings"
)

// spaHandler serves the built React app. Requests for a file that actually
// exists (index.html, /admin/assets/...) are served as-is; anything else
// (client-side routes like /admin/tokens) falls back to index.html so
// react-router can take over. The fallback serves the file's bytes
// directly rather than delegating to http.FileServerFS, since FileServer
// has a special case that 301-redirects any path ending in "/index.html"
// to its parent directory — which would silently rewrite the browser's
// address bar back to "/admin/" and break deep links/refresh on a route.
func (s *Server) spaHandler() http.HandlerFunc {
	fileServer := http.StripPrefix("/admin", http.FileServerFS(s.uiFS))

	return func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/admin"), "/")
		if reqPath == "" {
			reqPath = "index.html"
		}

		if _, err := fs.Stat(s.uiFS, reqPath); err != nil {
			s.serveIndex(w)
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}

func (s *Server) serveIndex(w http.ResponseWriter) {
	data, err := fs.ReadFile(s.uiFS, "index.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
