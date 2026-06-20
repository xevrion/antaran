package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type DevProcess struct {
	PID        uint32
	Name       string
	Cmdline    string
	Ports      []uint16
	RSS        uint64 // bytes
	StartTime  time.Time
	Uptime     time.Duration
}

func (p DevProcess) MemoryMB() float64 {
	return float64(p.RSS) / (1024 * 1024)
}

var watchNames = map[string]bool{}

func SetWatchList(names []string) {
	watchNames = make(map[string]bool, len(names))
	for _, n := range names {
		watchNames[n] = true
	}
}

func Scan(watchList []string) ([]DevProcess, error) {
	watched := make(map[string]bool, len(watchList))
	for _, n := range watchList {
		watched[n] = true
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	ports, _ := scanPorts()

	var procs []DevProcess
	for _, e := range entries {
		pid64, err := strconv.ParseUint(e.Name(), 10, 32)
		if err != nil {
			continue // not a PID directory
		}
		pid := uint32(pid64)

		name, err := readProcName(pid)
		if err != nil {
			continue
		}
		if !watched[name] {
			continue
		}

		cmdline, _ := readProcCmdline(pid)
		rss, _ := readProcRSS(pid)
		start, _ := readProcStartTime(pid)

		dp := DevProcess{
			PID:     pid,
			Name:    name,
			Cmdline: cmdline,
			RSS:     rss,
			Ports:   ports[pid],
		}
		if !start.IsZero() {
			dp.StartTime = start
			dp.Uptime = time.Since(start)
		}
		procs = append(procs, dp)
	}
	return procs, nil
}

func readProcName(pid uint32) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func readProcCmdline(pid uint32) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", err
	}
	// cmdline is NUL-delimited
	parts := strings.Split(string(data), "\x00")
	var clean []string
	for _, p := range parts {
		if p != "" {
			clean = append(clean, p)
		}
	}
	return strings.Join(clean, " "), nil
}

func readProcRSS(pid uint32) (uint64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseUint(fields[1], 10, 64)
				if err == nil {
					return kb * 1024, nil
				}
			}
		}
	}
	return 0, nil
}

func readProcStartTime(pid uint32) (time.Time, error) {
	statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return time.Time{}, err
	}

	uptimeData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return time.Time{}, err
	}

	// /proc/uptime: "uptime_seconds idle_seconds"
	uptimeFields := strings.Fields(string(uptimeData))
	if len(uptimeFields) < 1 {
		return time.Time{}, fmt.Errorf("unexpected /proc/uptime format")
	}
	uptimeSec, err := strconv.ParseFloat(uptimeFields[0], 64)
	if err != nil {
		return time.Time{}, err
	}

	// /proc/<pid>/stat: field 22 (0-indexed) is starttime in clock ticks
	// The comm field (2nd) may contain spaces and parens, so find closing ')' first
	statStr := string(statData)
	rparen := strings.LastIndex(statStr, ")")
	if rparen < 0 {
		return time.Time{}, fmt.Errorf("malformed /proc/%d/stat", pid)
	}
	fields := strings.Fields(statStr[rparen+1:])
	// field index 22 in the full stat is at index 20 after comm
	if len(fields) < 20 {
		return time.Time{}, fmt.Errorf("too few fields in /proc/%d/stat", pid)
	}
	startTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	clkTck := float64(100) // sysconf(_SC_CLK_TCK) is almost always 100 on Linux
	startSecAfterBoot := float64(startTicks) / clkTck
	processUptimeSec := uptimeSec - startSecAfterBoot

	startTime := time.Now().Add(-time.Duration(processUptimeSec * float64(time.Second)))
	return startTime, nil
}

func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", h, m)
}

func (p DevProcess) UptimeString() string {
	if p.Uptime == 0 {
		return "unknown"
	}
	return formatUptime(p.Uptime)
}

// ExeDir returns the working directory of the process if readable.
func ExeDir(pid uint32) string {
	link, err := filepath.EvalSymlinks(fmt.Sprintf("/proc/%d/cwd", pid))
	if err != nil {
		return ""
	}
	return link
}
