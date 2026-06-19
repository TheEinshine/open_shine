// Package web exposes the JSON admin API (and optionally serves the built SPA).
package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/TheEinshine/open_shine/auth"
	"github.com/TheEinshine/open_shine/db"
	"github.com/TheEinshine/open_shine/mailer"
)

// Server holds the API dependencies.
type Server struct {
	store     *db.Store
	auth      *auth.Authenticator
	staticDir string // optional: serve the built SPA from here (empty = API only)
	smtp      *mailer.Config
}

func New(store *db.Store, a *auth.Authenticator, staticDir string, smtp *mailer.Config) *Server {
	return &Server{store: store, auth: a, staticDir: staticDir, smtp: smtp}
}

// Handler builds the routed, middleware-wrapped HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public endpoints.
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /api/login", s.handleLogin)

	// Protected API: everything else under /api/ requires a session (+ CSRF on
	// mutating methods), enforced by RequireAuth.
	api := http.NewServeMux()
	api.HandleFunc("POST /api/logout", s.handleLogout)
	api.HandleFunc("GET /api/me", s.handleMe)
	api.HandleFunc("GET /api/metrics", s.handleMetrics)
	api.HandleFunc("GET /api/metrics/history", s.handleHistory)
	api.HandleFunc("GET /api/settings/mail", s.handleGetMail)
	api.HandleFunc("PUT /api/settings/mail", s.handlePutMail)
	api.HandleFunc("GET /api/thresholds", s.handleListThresholds)
	api.HandleFunc("POST /api/thresholds", s.handleCreateThreshold)
	api.HandleFunc("PUT /api/thresholds/{id}", s.handleUpdateThreshold)
	api.HandleFunc("DELETE /api/thresholds/{id}", s.handleDeleteThreshold)
	api.HandleFunc("GET /api/targets", s.handleListTargets)
	api.HandleFunc("POST /api/targets", s.handleCreateTarget)
	api.HandleFunc("PUT /api/targets/{id}", s.handleUpdateTarget)
	api.HandleFunc("DELETE /api/targets/{id}", s.handleDeleteTarget)
	api.HandleFunc("GET /api/logs", s.handleLogs)
	api.HandleFunc("GET /api/alerts", s.handleAlerts)
	api.HandleFunc("GET /api/newsletters", s.handleListNewsletters)
	api.HandleFunc("GET /api/newsletters/{id}", s.handleGetNewsletter)
	api.HandleFunc("POST /api/newsletters", s.handleCreateNewsletter)
	api.HandleFunc("PUT /api/newsletters/{id}", s.handleUpdateNewsletter)
	api.HandleFunc("DELETE /api/newsletters/{id}", s.handleDeleteNewsletter)
	api.HandleFunc("POST /api/newsletters/{id}/send", s.handleSendNewsletter)
	api.HandleFunc("GET /api/subscribers", s.handleListSubscribers)
	api.HandleFunc("POST /api/subscribers", s.handleAddSubscriber)
	api.HandleFunc("DELETE /api/subscribers/{id}", s.handleDeleteSubscriber)
	mux.Handle("/api/", s.auth.RequireAuth(api))

	// Optional SPA serving (Caddy normally does this in production).
	if s.staticDir != "" {
		mux.Handle("/", s.spaHandler())
	}

	return securityHeaders(mux)
}

// spaHandler serves static files, falling back to index.html for client-side
// routes (anything without a file extension that isn't an asset).
func (s *Server) spaHandler() http.Handler {
	fs := http.FileServer(http.Dir(s.staticDir))
	index := filepath.Join(s.staticDir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(r.URL.Path)
		if _, err := os.Stat(filepath.Join(s.staticDir, clean)); err == nil && clean != "/" {
			fs.ServeHTTP(w, r)
			return
		}
		if strings.Contains(filepath.Base(clean), ".") {
			http.NotFound(w, r) // missing asset, don't serve index for it
			return
		}
		http.ServeFile(w, r, index)
	})
}

// securityHeaders adds baseline hardening headers to every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		// HSTS is safe to send; it only takes effect over HTTPS.
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}
