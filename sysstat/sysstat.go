// Package sysstat collects host and Go-runtime metrics for the heartbeat
// report. Host metrics (CPU, memory, disk, load, uptime) are read straight from
// the Linux /proc filesystem and syscalls — no third-party dependency. On any
// non-Linux build (e.g. a Windows dev machine) host metrics are reported as
// unavailable and only the cross-platform runtime metrics are filled in.
package sysstat

import (
	"os"
	"runtime"
	"time"
)

// Stats is a point-in-time snapshot of host and runtime metrics.
type Stats struct {
	Time     time.Time
	Hostname string

	// Host metrics — valid only when HostAvailable is true.
	HostAvailable bool
	HostUp        time.Duration
	CPUPercent    float64 // 0..100; -1 when unavailable
	MemUsed       uint64
	MemTotal      uint64
	DiskUsed      uint64
	DiskTotal     uint64
	Load1         float64
	Load5         float64
	Load15        float64

	// Runtime metrics — always available.
	GoVersion  string
	Goroutines int
	HeapAlloc  uint64
}

// MemPercent returns memory used as a 0..100 percentage, or -1 if unknown.
func (s Stats) MemPercent() float64 {
	if s.MemTotal == 0 {
		return -1
	}
	return float64(s.MemUsed) / float64(s.MemTotal) * 100
}

// DiskPercent returns disk used as a 0..100 percentage, or -1 if unknown.
func (s Stats) DiskPercent() float64 {
	if s.DiskTotal == 0 {
		return -1
	}
	return float64(s.DiskUsed) / float64(s.DiskTotal) * 100
}

// Collect gathers a fresh snapshot. It blocks briefly (~200ms) on Linux while
// sampling CPU utilisation across a short window.
func Collect() Stats {
	s := Stats{
		Time:       time.Now(),
		CPUPercent: -1,
		GoVersion:  runtime.Version(),
		Goroutines: runtime.NumGoroutine(),
	}
	if h, err := os.Hostname(); err == nil {
		s.Hostname = h
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	s.HeapAlloc = m.HeapAlloc

	collectHost(&s) // platform-specific; fills host metrics or leaves them zero
	return s
}
