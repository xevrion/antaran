# AGENTS.md — Antaran Project Knowledge Base

> `CLAUDE.md` is a symlink to this file. Edit `AGENTS.md` only.

## Project Identity

**Antaran** (अंतरण — "transfer/handover" in Hindi/Sanskrit) — a native Linux
system tray daemon that watches your dev folder and surfaces what's actually
consuming your machine and attention.

- **Tagline**: knows what your dev folder is hiding
- **Stack**: Go 1.21+ (core) + Wails v2 (tray UI)
- **Platform**: Linux-first (Hyprland/Wayland primary, X11 graceful fallback)
- **Binary size target**: <15MB

## Repository Structure

```
antaran/
├── cmd/antaran/           # CLI entrypoint (main.go)
├── internal/
│   ├── config/            # Config loading (antaran.toml)
│   ├── scanner/           # Git repo watcher
│   │   ├── scanner.go     # Watcher interface + registry
│   │   ├── dirty.go       # Uncommitted changes detector
│   │   ├── unpushed.go    # Unpushed commits detector
│   │   └── stale.go       # Stale branch detector
│   └── process/           # Dev process watcher
│       ├── process.go     # /proc scanner
│       └── ports.go       # Port binding detector (/proc/net/tcp)
├── frontend/              # Wails web frontend (HTML/CSS/JS)
│   ├── src/
│   └── index.html
├── assets/                # Icons, branding
├── docs/                  # Extended documentation
│   ├── watchers.md        # Watcher interface docs
│   ├── config.md          # Config reference
│   └── faq.md             # Troubleshooting
├── .github/
│   ├── workflows/         # CI + release automation
│   └── ISSUE_TEMPLATE/    # Bug + feature templates
├── antaran.toml.example   # Annotated config example
└── Makefile               # Common dev tasks
```

## Development Commands

| Command                        | Purpose                              |
| ------------------------------ | ------------------------------------ |
| `go run ./cmd/antaran`         | Run CLI (no Wails required)          |
| `go build -o bin/antaran ./cmd/antaran` | Build CLI binary              |
| `go test ./...`                | Run all tests                        |
| `go vet ./...`                 | Static analysis                      |
| `wails dev`                    | Wails hot-reload dev mode            |
| `wails build`                  | Build full tray app binary           |
| `make fmt`                     | Run gofmt + goimports                |
| `make lint`                    | Run golangci-lint                    |

## Task Intake

Prefer requests with:

- `Goal`: exact bug, feature, or refactor target
- `Scope`: which package(s) to inspect first
- `Repro`: command or test that demonstrates the behavior
- `Expected` / `Actual`: what should happen vs. what does
- `Constraints`: what must not change

When scope is unclear, inspect in this order:

1. `cmd/antaran/main.go` — entrypoint and flag wiring
2. `internal/config/` — config schema and defaults
3. `internal/scanner/` — git repo watchers
4. `internal/process/` — /proc-based process scanner
5. `frontend/` — only for UI/tray rendering issues
6. `.github/workflows/` — only for CI/release issues

## Code Conventions

- All Go code must pass `gofmt` and `go vet` — CI rejects anything else
- Error strings: lowercase, no trailing punctuation
- All `/proc/<pid>/` reads must handle `ENOENT` gracefully — processes die mid-scan
- No `fmt.Println` in production paths — use the structured logger in `internal/log/`
- Comments only for non-obvious constraints or workarounds; never restate identifiers
- Tests live next to their source file (`foo_test.go` in same package)

## Key Invariants

- The scanner must never block the tray UI. All scans run in a goroutine; results are sent over a channel with a timeout.
- The process watcher reads `/proc` directly — no `ps`, no shell subprocesses. This keeps it fast and dependency-free.
- Git operations use `os/exec` with a hard 5-second timeout per repo. A hung `git status` must not hang the whole scan.
- Config is optional: if `~/.config/antaran/antaran.toml` does not exist, Antaran runs with sensible defaults (`~/Coding`, 30-second scan interval).

## Current Risk Areas

- `/proc/net/tcp` and `/proc/net/tcp6` encode ports in hex little-endian. Parsing these is subtle — see `internal/process/ports.go` for the canonical parser. Do not reimplement this logic elsewhere.
- Stale branch detection requires a `git fetch` to get remote state. This is the only network call Antaran makes. It's opt-in (disabled by default) because it can be slow on large repos or slow networks.
- The Wails frontend communicates with Go via `wails.Bind()`. The bound struct is `App` in `cmd/antaran/app.go`. When adding a new method, keep it on `App` — don't create parallel bound types.
- `libwebkit2gtk` version requirements vary by distro. On Ubuntu 22.04 you need `libwebkit2gtk-4.0-dev`; on Ubuntu 24.04 it's `libwebkit2gtk-4.1-dev`. The CI matrix covers both.

## Branch Strategy

- `main` — only branch. All development and releases happen here.
- Tag format: `v0.x.x` (lowercase v). Current version: check `go.mod` module line or `cmd/antaran/version.go`.

## Release Workflow (CI)

Pushing a `v*` tag triggers `.github/workflows/release.yml`:

1. **build** — compiles for `linux/amd64` and `linux/arm64`
2. **package** — produces `.tar.gz` archives + checksums
3. **release** — creates GitHub Release and attaches artifacts

`.github/workflows/ci.yml` runs on every push and PR: `go test`, `go vet`, `gofmt` check.

## Watcher Interface

Adding a new watcher means implementing:

```go
type Watcher interface {
    Scan(ctx context.Context) ([]Finding, error)
    Name() string
}
```

Register it in `internal/scanner/registry.go`. See `docs/watchers.md` for the
full Finding schema.

## Documentation Guidelines

- **README.md**: features, install, quick start, screenshot. Keep it short.
- **docs/config.md**: every config key, type, default, and example.
- **docs/watchers.md**: watcher interface, Finding schema, how to add one.
- **docs/faq.md**: distro-specific issues, permission problems, Wails build deps.
- Don't put config key details in the README — link to `docs/config.md`.
