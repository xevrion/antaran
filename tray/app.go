package tray

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/xevrion/antaran/internal/config"
	"github.com/xevrion/antaran/internal/process"
	"github.com/xevrion/antaran/internal/scanner"
)

// App is the struct bound to the Wails frontend via wails.Bind().
// All methods on this type are callable from JavaScript.
type App struct {
	mu      sync.RWMutex
	cfg     *config.Config
	repos   []scanner.RepoStatus
	procs   []process.DevProcess
	summary string
}

func NewApp(cfg *config.Config) *App {
	return &App{cfg: cfg}
}

// --- called by the tray daemon goroutine ---

func (a *App) UpdateData(repos []scanner.RepoStatus, procs []process.DevProcess) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.repos = repos
	a.procs = procs
	a.summary = buildSummary(repos, procs)
}

func (a *App) Summary() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.summary
}

// --- Wails-bound methods (callable from JS) ---

type RepoView struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Branch         string `json:"branch"`
	StagedCount    int    `json:"staged"`
	UnstagedCount  int    `json:"unstaged"`
	UntrackedCount int    `json:"untracked"`
	UnpushedCount  int    `json:"unpushed"`
	StaleDays      int    `json:"stale_days"`
	Flags          string `json:"flags"`
}

type ProcessView struct {
	PID        uint32   `json:"pid"`
	Name       string   `json:"name"`
	Cmdline    string   `json:"cmdline"`
	Uptime     string   `json:"uptime"`
	MemoryMB   float64  `json:"memory_mb"`
	Ports      []uint16 `json:"ports"`
	KillLabel  string   `json:"kill_label"`
}

type ScanResult struct {
	Summary   string        `json:"summary"`
	Repos     []RepoView    `json:"repos"`
	Processes []ProcessView `json:"processes"`
}

func (a *App) GetScanResult() ScanResult {
	a.mu.RLock()
	defer a.mu.RUnlock()

	repos := make([]RepoView, 0, len(a.repos))
	for _, r := range a.repos {
		if !r.Dirty && r.UnpushedCount == 0 && r.Staledays < a.cfg.Git.StaleAfterDays {
			continue
		}
		repos = append(repos, RepoView{
			Name:           r.Name,
			Path:           r.Path,
			Branch:         r.CurrentBranch,
			StagedCount:    r.StagedCount,
			UnstagedCount:  r.UnstagedCount,
			UntrackedCount: r.UntrackedCount,
			UnpushedCount:  r.UnpushedCount,
			StaleDays:      r.Staledays,
			Flags:          repoFlags(r, a.cfg.Git.StaleAfterDays),
		})
	}

	procs := make([]ProcessView, 0, len(a.procs))
	for _, p := range a.procs {
		procs = append(procs, ProcessView{
			PID:       p.PID,
			Name:      p.Name,
			Cmdline:   p.Cmdline,
			Uptime:    p.UptimeString(),
			MemoryMB:  p.MemoryMB(),
			Ports:     p.Ports,
			KillLabel: fmt.Sprintf("kill %s pid:%d", p.Name, p.PID),
		})
	}

	return ScanResult{
		Summary:   a.summary,
		Repos:     repos,
		Processes: procs,
	}
}

// KillProcess sends SIGTERM (then SIGKILL) to the given PID.
// Returns a human-readable result string shown in the UI.
func (a *App) KillProcess(pid uint32, name string) string {
	result := process.Kill(pid, name)
	if result.Success {
		return fmt.Sprintf("killed %s (pid %d) with %s", name, pid, result.Signal)
	}
	return fmt.Sprintf("failed to kill %s (pid %d): %v", name, pid, result.Error)
}

// OpenInEditor opens the repo path in the user's $EDITOR or xdg-open.
func (a *App) OpenInEditor(path string) string {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		if editor := findEditor(); editor != "" {
			cmd = exec.Command(editor, path)
		} else {
			cmd = exec.Command("xdg-open", path)
		}
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Sprintf("failed to open editor: %v", err)
	}
	return "opened"
}

// RefreshNow triggers an immediate rescan and returns the updated result.
func (a *App) RefreshNow() ScanResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sc := scanner.New(a.cfg.ScanRoot, a.cfg.Git.MaxDepth)
	repos, _ := sc.Scan(ctx)
	procs, _ := process.Scan(a.cfg.Process.Watch)
	a.UpdateData(repos, procs)
	return a.GetScanResult()
}

// --- helpers ---

func buildSummary(repos []scanner.RepoStatus, procs []process.DevProcess) string {
	dirtyCount := 0
	for _, r := range repos {
		if r.Dirty || r.UnpushedCount > 0 {
			dirtyCount++
		}
	}
	totalMem := 0.0
	for _, p := range procs {
		totalMem += p.MemoryMB()
	}

	var parts []string
	if dirtyCount > 0 {
		s := fmt.Sprintf("%d dirty repo", dirtyCount)
		if dirtyCount != 1 {
			s += "s"
		}
		parts = append(parts, s)
	}
	if len(procs) > 0 {
		s := fmt.Sprintf("%d zombie dev server", len(procs))
		if len(procs) != 1 {
			s += "s"
		}
		if totalMem > 0 {
			s += fmt.Sprintf(" eating %.0fMB", totalMem)
		}
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return "everything looks clean"
	}
	return strings.Join(parts, " · ")
}

func repoFlags(r scanner.RepoStatus, staleDays int) string {
	var flags []string
	if r.StagedCount > 0 {
		flags = append(flags, fmt.Sprintf("%d staged", r.StagedCount))
	}
	if r.UnstagedCount > 0 {
		flags = append(flags, fmt.Sprintf("%d unstaged", r.UnstagedCount))
	}
	if r.UntrackedCount > 0 {
		flags = append(flags, fmt.Sprintf("%d untracked", r.UntrackedCount))
	}
	if r.UnpushedCount > 0 {
		flags = append(flags, fmt.Sprintf("%d unpushed", r.UnpushedCount))
	}
	if r.Staledays >= staleDays {
		flags = append(flags, fmt.Sprintf("stale %dd", r.Staledays))
	}
	if len(flags) == 0 {
		return "clean"
	}
	return strings.Join(flags, ", ")
}

func findEditor() string {
	for _, e := range []string{"code", "zed", "hx", "nvim", "vim", "gedit"} {
		if p, err := exec.LookPath(e); err == nil && p != "" {
			return p
		}
	}
	return ""
}
