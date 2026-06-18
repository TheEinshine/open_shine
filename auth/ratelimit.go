package auth

import (
	"sync"
	"time"
)

// limiter is an in-memory, per-IP login throttle. After max consecutive
// failures an IP is locked out for lockFor. It's intentionally simple and
// process-local (fine for a single-instance server).
type limiter struct {
	mu      sync.Mutex
	entries map[string]*entry
	max     int
	lockFor time.Duration
}

type entry struct {
	fails       int
	lockedUntil time.Time
	seen        time.Time
}

func newLimiter(max int, lockFor time.Duration) *limiter {
	return &limiter{entries: map[string]*entry{}, max: max, lockFor: lockFor}
}

// locked reports whether ip is currently locked out.
func (l *limiter) locked(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.entries[ip]
	return e != nil && time.Now().Before(e.lockedUntil)
}

// fail records a failed attempt and locks the IP once max is reached.
func (l *limiter) fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.gc()
	e := l.entries[ip]
	if e == nil {
		e = &entry{}
		l.entries[ip] = e
	}
	e.fails++
	e.seen = time.Now()
	if e.fails >= l.max {
		e.lockedUntil = time.Now().Add(l.lockFor)
		e.fails = 0
	}
}

// reset clears the counter after a successful login.
func (l *limiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, ip)
}

// gc drops stale entries; caller holds the lock.
func (l *limiter) gc() {
	cutoff := time.Now().Add(-l.lockFor - time.Hour)
	for ip, e := range l.entries {
		if e.seen.Before(cutoff) && time.Now().After(e.lockedUntil) {
			delete(l.entries, ip)
		}
	}
}
