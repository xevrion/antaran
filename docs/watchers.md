# Watcher Modules

Antaran's scanning logic is split into two packages:

- `internal/scanner/` — git repository watchers
- `internal/process/` — dev process watchers

## RepoStatus (scanner output)

```go
type RepoStatus struct {
    Path           string  // absolute path to repo root
    Name           string  // directory name
    Dirty          bool    // any uncommitted changes
    StagedCount    int     // files staged for commit
    UnstagedCount  int     // tracked files with unstaged changes
    UntrackedCount int     // untracked files
    UnpushedCount  int     // commits ahead of origin/<branch>
    CurrentBranch  string  // output of git rev-parse --abbrev-ref HEAD
    Staledays      int     // days since last commit (0 = recent)
}
```

## DevProcess (process output)

```go
type DevProcess struct {
    PID       uint32
    Name      string    // from /proc/<pid>/comm
    Cmdline   string    // from /proc/<pid>/cmdline (NUL-delimited, joined)
    Ports     []uint16  // TCP ports the process is listening on
    RSS       uint64    // resident set size in bytes
    StartTime time.Time
    Uptime    time.Duration
}
```

## Adding a Git Watcher

1. Create `internal/scanner/mywatcher.go`
2. Add a function with signature:
   ```go
   func myCheck(ctx context.Context, repoPath string, status *RepoStatus) error
   ```
3. Call it from `inspectRepo()` in `internal/scanner/git.go`
4. Add a field to `RepoStatus` if you need to surface new data
5. Render it in `cmd/antaran/main.go` → `repoFlags()`

## Adding a Process Watcher

The process scanner in `internal/process/process.go` works by:

1. Reading every numeric directory under `/proc`
2. Reading `/proc/<pid>/comm` and matching against the configured watch list
3. For matched processes, reading RSS from `/proc/<pid>/status`, uptime from `/proc/<pid>/stat`, and ports from `/proc/net/tcp` + `/proc/<pid>/fd/`

To watch a new kind of process, add its `comm` name to the `watch` list in `antaran.toml` — no code change needed.

To add a new piece of per-process data (e.g. CPU%), add a field to `DevProcess`, populate it in `Scan()`, and render it in `cmd/antaran/main.go`.

## Port Detection Notes

Port detection reads `/proc/net/tcp` and `/proc/net/tcp6`, filtering for state `0A` (TCP_LISTEN). The local address field is hex-encoded with the port in the last 4 characters (big-endian). See `internal/process/ports.go` for the canonical parser — do not reimplement this logic.
