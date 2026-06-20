package process

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type KillResult struct {
	PID        uint32
	Name       string
	Signal     syscall.Signal
	Success    bool
	Error      error
}

// Kill sends SIGTERM to pid, waits up to 2s, then SIGKILL if still alive.
// Every call — success or failure — is written to the audit log before any
// signal is sent. Callers must never kill without a log entry existing first.
func Kill(pid uint32, name string) KillResult {
	logKillAttempt(pid, name, syscall.SIGTERM)

	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return KillResult{PID: pid, Name: name, Signal: syscall.SIGTERM, Error: err}
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		logKillResult(pid, name, syscall.SIGTERM, false, err)
		return KillResult{PID: pid, Name: name, Signal: syscall.SIGTERM, Error: err}
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !procAlive(pid) {
			logKillResult(pid, name, syscall.SIGTERM, true, nil)
			return KillResult{PID: pid, Name: name, Signal: syscall.SIGTERM, Success: true}
		}
		time.Sleep(100 * time.Millisecond)
	}

	// SIGTERM didn't work in 2s — escalate
	logKillAttempt(pid, name, syscall.SIGKILL)
	if err := proc.Signal(syscall.SIGKILL); err != nil {
		logKillResult(pid, name, syscall.SIGKILL, false, err)
		return KillResult{PID: pid, Name: name, Signal: syscall.SIGKILL, Error: err}
	}
	logKillResult(pid, name, syscall.SIGKILL, true, nil)
	return KillResult{PID: pid, Name: name, Signal: syscall.SIGKILL, Success: true}
}

func procAlive(pid uint32) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

func logDir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "antaran")
}

func logKillAttempt(pid uint32, name string, sig syscall.Signal) {
	writeLog(fmt.Sprintf("KILL_ATTEMPT pid=%d name=%s signal=%s", pid, name, sig))
}

func logKillResult(pid uint32, name string, sig syscall.Signal, ok bool, err error) {
	status := "success"
	if !ok {
		status = fmt.Sprintf("failed: %v", err)
	}
	writeLog(fmt.Sprintf("KILL_RESULT  pid=%d name=%s signal=%s status=%s", pid, name, sig, status))
}

func writeLog(msg string) {
	dir := logDir()
	_ = os.MkdirAll(dir, 0o700)
	f, err := os.OpenFile(filepath.Join(dir, "operations.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s  %s\n", time.Now().UTC().Format(time.RFC3339), msg)
}
