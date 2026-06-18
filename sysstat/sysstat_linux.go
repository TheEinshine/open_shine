//go:build linux

package sysstat

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// collectHost fills the host metrics on Linux. Each reader fails soft: an
// unreadable source leaves its field at the zero value rather than erroring,
// so a partial /proc still yields a useful report.
func collectHost(s *Stats) {
	s.HostAvailable = true
	if p, ok := cpuPercent(); ok {
		s.CPUPercent = p
	}
	s.MemUsed, s.MemTotal = readMem()
	s.DiskUsed, s.DiskTotal = readDisk("/")
	s.HostUp = readUptime()
	s.Load1, s.Load5, s.Load15 = readLoad()
}

// cpuPercent samples /proc/stat twice over a short window and returns the busy
// percentage across all cores.
func cpuPercent() (float64, bool) {
	total1, idle1, ok := cpuSample()
	if !ok {
		return 0, false
	}
	time.Sleep(200 * time.Millisecond)
	total2, idle2, ok := cpuSample()
	if !ok || total2 <= total1 {
		return 0, false
	}
	totalDelta := float64(total2 - total1)
	idleDelta := float64(idle2 - idle1)
	return (1 - idleDelta/totalDelta) * 100, true
}

// cpuSample parses the aggregate "cpu" line of /proc/stat into total and idle
// jiffies (idle = idle + iowait).
func cpuSample() (total, idle uint64, ok bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	line, _, _ := strings.Cut(string(data), "\n")
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, false
	}
	for i := 1; i < len(fields); i++ {
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, 0, false
		}
		total += v
		if i == 4 || i == 5 { // idle, iowait
			idle += v
		}
	}
	return total, idle, true
}

// readMem returns used and total RAM in bytes from /proc/meminfo. "Used" is
// Total minus Available (the kernel's estimate of reclaimable memory).
func readMem() (used, total uint64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	var memTotal, memAvail uint64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		v, err := strconv.ParseUint(fields[1], 10, 64) // kB
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			memTotal = v * 1024
		case "MemAvailable:":
			memAvail = v * 1024
		}
	}
	if memTotal == 0 {
		return 0, 0
	}
	if memAvail > memTotal {
		memAvail = memTotal
	}
	return memTotal - memAvail, memTotal
}

// readDisk returns used and total bytes for the filesystem backing path.
// "Used" is computed against space available to unprivileged users (Bavail).
func readDisk(path string) (used, total uint64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	bs := uint64(st.Bsize)
	total = st.Blocks * bs
	free := st.Bavail * bs
	if free > total {
		free = total
	}
	return total - free, total
}

// readUptime returns host uptime from /proc/uptime.
func readUptime() time.Duration {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	first, _, _ := strings.Cut(strings.TrimSpace(string(data)), " ")
	secs, err := strconv.ParseFloat(first, 64)
	if err != nil {
		return 0
	}
	return time.Duration(secs * float64(time.Second))
}

// readLoad returns the 1/5/15-minute load averages from /proc/loadavg.
func readLoad() (l1, l5, l15 float64) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0
	}
	l1, _ = strconv.ParseFloat(fields[0], 64)
	l5, _ = strconv.ParseFloat(fields[1], 64)
	l15, _ = strconv.ParseFloat(fields[2], 64)
	return l1, l5, l15
}
