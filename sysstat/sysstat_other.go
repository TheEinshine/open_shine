//go:build !linux

package sysstat

// collectHost is the fallback for non-Linux builds (e.g. a Windows dev
// machine). Host metrics aren't gathered here; only the runtime metrics filled
// in by Collect are reported. The production target is Linux.
func collectHost(s *Stats) {
	s.HostAvailable = false
}
