package web

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/TheEinshine/open_shine/auth"
	"github.com/TheEinshine/open_shine/db"
	"github.com/TheEinshine/open_shine/newsletter"
	"github.com/TheEinshine/open_shine/sysstat"
)

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---- auth ----

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	user, err := s.auth.Login(w, r, in.Email, in.Password)
	switch err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]any{"user": user})
	case auth.ErrInvalid:
		writeError(w, http.StatusUnauthorized, err.Error())
	case auth.ErrLocked:
		writeError(w, http.StatusTooManyRequests, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "login failed")
	}
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.auth.Logout(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	sess, _ := auth.SessionFrom(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "csrf": sess.CSRFToken})
}

// ---- metrics ----

type usageDTO struct {
	Used    uint64  `json:"used"`
	Total   uint64  `json:"total"`
	Percent float64 `json:"percent"`
}

type metricsDTO struct {
	Time          time.Time  `json:"time"`
	Host          string     `json:"host"`
	HostAvailable bool       `json:"hostAvailable"`
	UptimeSeconds float64    `json:"uptimeSeconds"`
	CPU           float64    `json:"cpu"`
	Mem           usageDTO   `json:"mem"`
	Disk          usageDTO   `json:"disk"`
	Load          [3]float64 `json:"load"`
	Go            struct {
		Version    string `json:"version"`
		Goroutines int    `json:"goroutines"`
		HeapBytes  uint64 `json:"heapBytes"`
	} `json:"go"`
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	st := sysstat.Collect()
	var d metricsDTO
	d.Time = st.Time
	d.Host = st.Hostname
	d.HostAvailable = st.HostAvailable
	d.UptimeSeconds = st.HostUp.Seconds()
	d.CPU = st.CPUPercent
	d.Mem = usageDTO{Used: st.MemUsed, Total: st.MemTotal, Percent: st.MemPercent()}
	d.Disk = usageDTO{Used: st.DiskUsed, Total: st.DiskTotal, Percent: st.DiskPercent()}
	d.Load = [3]float64{st.Load1, st.Load5, st.Load15}
	d.Go.Version = st.GoVersion
	d.Go.Goroutines = st.Goroutines
	d.Go.HeapBytes = st.HeapAlloc
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 120, 1, 1000)
	points, err := s.store.MetricHistory(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read history")
		return
	}
	writeJSON(w, http.StatusOK, points)
}

// ---- mail settings ----

func (s *Server) handleGetMail(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read settings")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handlePutMail(w http.ResponseWriter, r *http.Request) {
	var in db.Settings
	if !readJSON(w, r, &in) {
		return
	}
	if in.IntervalMins < 1 {
		in.IntervalMins = 10
	}
	if in.Subject == "" {
		in.Subject = "Open Shine heartbeat"
	}
	if err := s.store.UpdateSettings(in); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save settings")
		return
	}
	writeJSON(w, http.StatusOK, in)
}

// ---- thresholds ----

var validMetrics = map[string]bool{"cpu": true, "mem": true, "disk": true, "load1": true}
var validOps = map[string]bool{"gt": true, "gte": true, "lt": true, "lte": true}

func (s *Server) handleListThresholds(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListThresholds(false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list thresholds")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateThreshold(w http.ResponseWriter, r *http.Request) {
	var in db.Threshold
	if !readJSON(w, r, &in) {
		return
	}
	if !validMetrics[in.Metric] || !validOps[in.Op] {
		writeError(w, http.StatusBadRequest, "metric must be cpu|mem|disk|load1 and op must be gt|gte|lt|lte")
		return
	}
	id, err := s.store.CreateThreshold(in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create threshold")
		return
	}
	in.ID = id
	writeJSON(w, http.StatusCreated, in)
}

func (s *Server) handleUpdateThreshold(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in db.Threshold
	if !readJSON(w, r, &in) {
		return
	}
	if !validMetrics[in.Metric] || !validOps[in.Op] {
		writeError(w, http.StatusBadRequest, "metric must be cpu|mem|disk|load1 and op must be gt|gte|lt|lte")
		return
	}
	in.ID = id
	if err := s.store.UpdateThreshold(in); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update threshold")
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func (s *Server) handleDeleteThreshold(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteThreshold(id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete threshold")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ---- targets ----

var validKinds = map[string]bool{"http": true, "tcp": true}

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListTargets(false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list targets")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateTarget(w http.ResponseWriter, r *http.Request) {
	var in db.Target
	if !readJSON(w, r, &in) {
		return
	}
	if in.Name == "" || in.Address == "" || !validKinds[in.Kind] {
		writeError(w, http.StatusBadRequest, "name, address required and kind must be http|tcp")
		return
	}
	id, err := s.store.CreateTarget(in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create target")
		return
	}
	in.ID = id
	writeJSON(w, http.StatusCreated, in)
}

func (s *Server) handleUpdateTarget(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in db.Target
	if !readJSON(w, r, &in) {
		return
	}
	if in.Name == "" || in.Address == "" || !validKinds[in.Kind] {
		writeError(w, http.StatusBadRequest, "name, address required and kind must be http|tcp")
		return
	}
	in.ID = id
	if err := s.store.UpdateTarget(in); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update target")
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func (s *Server) handleDeleteTarget(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteTarget(id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete target")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ---- logs / alerts ----

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50, 1, 500)
	logs, err := s.store.RecentLogs(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read logs")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50, 1, 500)
	alerts, err := s.store.RecentAlerts(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read alerts")
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}

// ---- newsletters ----

func (s *Server) handleListNewsletters(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListNewsletters()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list newsletters")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleGetNewsletter(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	nl, err := s.store.GetNewsletter(id)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "newsletter not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read newsletter")
		return
	}
	writeJSON(w, http.StatusOK, nl)
}

func (s *Server) handleCreateNewsletter(w http.ResponseWriter, r *http.Request) {
	var in db.Newsletter
	if !readJSON(w, r, &in) {
		return
	}
	if in.Title == "" || in.Subject == "" || in.Recipient == "" || in.BodyHTML == "" {
		writeError(w, http.StatusBadRequest, "title, subject, recipient, and bodyHtml are required")
		return
	}
	id, err := s.store.CreateNewsletter(in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create newsletter")
		return
	}
	in.ID = id
	writeJSON(w, http.StatusCreated, in)
}

func (s *Server) handleUpdateNewsletter(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in db.Newsletter
	if !readJSON(w, r, &in) {
		return
	}
	if in.Title == "" || in.Subject == "" || in.Recipient == "" || in.BodyHTML == "" {
		writeError(w, http.StatusBadRequest, "title, subject, recipient, and bodyHtml are required")
		return
	}
	in.ID = id
	if err := s.store.UpdateNewsletter(in); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update newsletter")
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func (s *Server) handleDeleteNewsletter(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteNewsletter(id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete newsletter")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleSendNewsletter(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if s.smtp == nil {
		writeError(w, http.StatusServiceUnavailable, "SMTP is not configured")
		return
	}
	if err := newsletter.SendNow(s.store, *s.smtp, id); err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "newsletter not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not send newsletter: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// ---- helpers ----

func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func queryInt(r *http.Request, key string, def, min, max int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

