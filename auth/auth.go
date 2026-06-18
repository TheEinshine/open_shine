package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/TheEinshine/open_shine/db"
)

const (
	sessionCookie = "sid"
	csrfCookie    = "csrf"
	csrfHeader    = "X-CSRF-Token"
)

// Public errors returned by Login.
var (
	ErrInvalid = errors.New("invalid email or password")
	ErrLocked  = errors.New("too many attempts, try again later")
)

// Config configures an Authenticator.
type Config struct {
	Store        *db.Store
	SessionTTL   time.Duration
	CookieSecure bool // set Secure on cookies (true in production behind TLS)
}

// Authenticator issues and validates login sessions.
type Authenticator struct {
	cfg     Config
	limiter *limiter
}

func New(cfg Config) *Authenticator {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 7 * 24 * time.Hour
	}
	return &Authenticator{cfg: cfg, limiter: newLimiter(5, 10*time.Minute)}
}

// Login verifies credentials and, on success, creates a session and sets the
// session + CSRF cookies. The returned user has no password hash.
func (a *Authenticator) Login(w http.ResponseWriter, r *http.Request, email, password string) (db.User, error) {
	ip := clientIP(r)
	if a.limiter.locked(ip) {
		return db.User{}, ErrLocked
	}

	u, err := a.cfg.Store.UserByEmail(strings.TrimSpace(email))
	if err != nil || u.PasswordHash == "" || !Verify(u.PasswordHash, password) {
		a.limiter.fail(ip)
		return db.User{}, ErrInvalid
	}
	a.limiter.reset(ip)

	sid, err := Token()
	if err != nil {
		return db.User{}, err
	}
	csrf, err := Token()
	if err != nil {
		return db.User{}, err
	}
	if err := a.cfg.Store.CreateSession(sid, u.ID, csrf, a.cfg.SessionTTL); err != nil {
		return db.User{}, err
	}

	a.setCookie(w, sessionCookie, sid, true)
	a.setCookie(w, csrfCookie, csrf, false) // readable by JS so the SPA can echo it
	u.PasswordHash = ""
	return u, nil
}

// Logout deletes the current session and clears cookies.
func (a *Authenticator) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = a.cfg.Store.DeleteSession(c.Value)
	}
	a.clearCookie(w, sessionCookie, true)
	a.clearCookie(w, csrfCookie, false)
}

// RequireAuth wraps a handler, rejecting requests without a valid session and
// enforcing CSRF on state-changing methods.
func (a *Authenticator) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, sess, ok := a.session(r)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if isMutating(r.Method) {
			if tok := r.Header.Get(csrfHeader); tok == "" || tok != sess.CSRFToken {
				writeErr(w, http.StatusForbidden, "invalid csrf token")
				return
			}
		}
		ctx := context.WithValue(r.Context(), userKey, u)
		ctx = context.WithValue(ctx, sessKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Authenticator) session(r *http.Request) (db.User, db.Session, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return db.User{}, db.Session{}, false
	}
	sess, err := a.cfg.Store.SessionByID(c.Value)
	if err != nil {
		return db.User{}, db.Session{}, false
	}
	u, err := a.cfg.Store.UserByID(sess.UserID)
	if err != nil {
		return db.User{}, db.Session{}, false
	}
	u.PasswordHash = ""
	return u, sess, true
}

func (a *Authenticator) setCookie(w http.ResponseWriter, name, val string, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    val,
		Path:     "/",
		HttpOnly: httpOnly,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(a.cfg.SessionTTL.Seconds()),
	})
}

func (a *Authenticator) clearCookie(w http.ResponseWriter, name string, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: httpOnly,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// EnsureAdmin creates the admin account from env if it doesn't already exist.
// Returns true if a new account was created.
func EnsureAdmin(store *db.Store, name, email, password string) (bool, error) {
	if email == "" || password == "" {
		return false, nil
	}
	switch _, err := store.UserByEmail(email); {
	case err == nil:
		return false, nil // already exists
	case errors.Is(err, sql.ErrNoRows):
		// create below
	default:
		return false, err
	}
	hash, err := Hash(password)
	if err != nil {
		return false, err
	}
	if _, err := store.CreateUser(name, email, hash); err != nil {
		return false, err
	}
	return true, nil
}

// ---- request context ----

type ctxKey int

const (
	userKey ctxKey = iota
	sessKey
)

// UserFrom returns the authenticated user attached by RequireAuth.
func UserFrom(ctx context.Context) (db.User, bool) {
	u, ok := ctx.Value(userKey).(db.User)
	return u, ok
}

// SessionFrom returns the session attached by RequireAuth.
func SessionFrom(ctx context.Context) (db.Session, bool) {
	s, ok := ctx.Value(sessKey).(db.Session)
	return s, ok
}

// ---- helpers ----

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// clientIP extracts the caller IP, trusting the first X-Forwarded-For hop set
// by the reverse proxy (Caddy) when present.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
