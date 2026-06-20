# Contributing to Antaran

Welcome — and thank you for wanting to make Antaran better.

Antaran is small by design. Before building something large, open a Discussion
to check it fits the project's direction. For typos, bug fixes, and obvious
improvements, a PR is enough — no issue required.

## Branch Strategy

All development happens directly on `main`. Submit pull requests to `main`.
There are no long-lived feature branches.

## Dev Setup

### Prerequisites

- Go ≥ 1.21
- `git` in PATH (used by the scanner at runtime)
- Linux (Wayland or X11). Hyprland is the primary test target.

For the tray UI (optional during core development):
- Wails v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- `libwebkit2gtk-4.0-dev`, `libgtk-3-dev`, `pkg-config`

### Building

```bash
# Clone
git clone https://github.com/xevrion/antaran.git
cd antaran

# Run the CLI (no Wails required)
go run ./cmd/antaran --help

# Build the CLI binary
go build -o bin/antaran ./cmd/antaran

# Run tests
go test ./...

# Build the full tray app (requires Wails)
wails build
```

### Config file

Antaran reads `~/.config/antaran/antaran.toml` on startup. Copy the example:

```bash
cp antaran.toml.example ~/.config/antaran/antaran.toml
```

## Adding a New Watcher Module

Watchers live in `internal/scanner/` and `internal/process/`. Each module
implements a small interface:

```go
type Watcher interface {
    Scan(ctx context.Context) ([]Finding, error)
    Name() string
}
```

1. Create `internal/scanner/yourwatcher.go`
2. Implement `Watcher`
3. Register it in `internal/scanner/registry.go`
4. Add a test in `internal/scanner/yourwatcher_test.go`
5. Document the new finding type in `docs/watchers.md`

Keep each watcher focused on one concern. A watcher that does two things
should be two watchers.

## Code Conventions

- `gofmt` before committing (CI enforces this)
- No comments that restate the function name — only write comments explaining
  non-obvious constraints or workarounds
- Error strings are lowercase and don't end with punctuation (Go convention)
- All `/proc` reads must handle the file vanishing mid-read — processes die

## Commit Messages

Follow conventional commits loosely:

```
feat: detect stale branches older than 14 days
fix: don't crash when /proc/<pid>/fd is unreadable
docs: add watcher interface documentation
```

One subject line, present tense, ≤72 characters. Body is optional but welcome
for non-obvious changes.

## Pull Request Checklist

- [ ] `go test ./...` passes
- [ ] `go vet ./...` clean
- [ ] New watcher has a test
- [ ] Config options documented in `antaran.toml.example`
- [ ] No `fmt.Println` left in production paths (use the logger)

## Questions

Open a [GitHub Discussion](https://github.com/xevrion/antaran/discussions).
For bugs, use the [bug report template](.github/ISSUE_TEMPLATE/bug-report.yml).
