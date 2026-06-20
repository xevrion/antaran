# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> `CLAUDE.md` is a symlink to this file. Edit `AGENTS.md` only.

## Project Identity

**Antaran** (अंतरण — "transfer/handover" in Hindi/Sanskrit) is a native Linux system tray daemon that watches your dev folder and surfaces what is actually consuming your machine and attention.

- **Tagline**: knows what your dev folder is hiding
- **Stack**: Go 1.21+ (core) + Wails v2 (tray UI) + pure-Go DBus SNI (tray icon)
- **Platform**: Linux-first (Hyprland/Wayland primary, X11 graceful fallback)

## Repository Structure

```
antaran/
├── cmd/antaran/           # CLI binary entrypoint
├── cmd/antaran-tray/      # Wails tray app entrypoint
│   ├── main.go            # Wires daemon + Wails window
│   ├── wails.json         # Wails project manifest
│   └── frontend/dist/     # Hand-written HTML/CSS/JS frontend (embedded at build)
├── tray/                  # Tray-specific library (imported by cmd/antaran-tray)
│   ├── app.go             # App struct bound to Wails — all JS-callable methods
│   ├── daemon.go          # Scan ticker + SNI icon lifecycle
│   ├── sni.go             # Pure-Go DBus StatusNotifierItem implementation
│   └── icon.go            # Embedded fallback icon bytes
├── internal/
│   ├── config/config.go   # TOML config loader with defaults
│   ├── scanner/           # Git repo scanning
│   │   ├── scanner.go     # walkGitRepos + RepoStatus type
│   │   └── git.go         # git status/rev-list/log via os/exec
│   └── process/           # Dev process detection
│       ├── process.go     # /proc scanner, RSS, uptime, cmdline
│       ├── ports.go       # /proc/net/tcp parser — listening port detection
│       └── kill.go        # SIGTERM→SIGKILL with audit log
├── docs/
│   ├── watchers.md        # RepoStatus + DevProcess schemas, how to extend
│   └── faq.md             # Distro-specific build and runtime issues
├── scripts/pkgconfig-shim.sh  # Fedora webkit2gtk-4.0 shim helper
├── antaran.toml.example   # Annotated config reference
└── Makefile
```

## Development Commands

```bash
# CLI (no Wails required)
make build              # builds bin/antaran
make run ARGS="--root ~/projects"
go test -race ./...
go vet ./...
gofmt -w .

# Tray app (requires Wails + libwebkit2gtk)
make pkgconfig-shim     # run once on Fedora 40+ to create webkit2gtk-4.0 shim
export PKG_CONFIG_PATH="$HOME/.cache/antaran-pkgconfig:$PKG_CONFIG_PATH"
make build-tray         # builds cmd/antaran-tray/build/bin/bin/antaran-tray
make run-tray           # GDK_BACKEND=x11 DISPLAY=:0 ... (required on Nvidia)

# Install
make install            # CLI to ~/.local/bin/antaran
make install-tray       # tray app to ~/.local/bin/antaran-tray
```

Single package test: `go test -race ./internal/scanner/`

## Architecture

### Two binaries, one library core

`cmd/antaran` is a pure CLI — no CGO, no Wails. It imports `internal/` directly and prints human or JSON output. Good for scripting and testing the scanning logic in isolation.

`cmd/antaran-tray` is the GUI binary. It imports `tray/` which wraps the same `internal/` packages. Wails embeds `cmd/antaran-tray/frontend/dist/index.html` at compile time via `//go:embed`. The embed path must be stripped with `fs.Sub(assetsFS, "frontend/dist")` before passing to Wails — Wails expects `index.html` at the FS root.

### Tray icon vs. Wails window

These are deliberately separate concerns:

- **Tray icon** (`tray/sni.go`): pure-Go DBus implementation of the StatusNotifierItem protocol. Registers on the session bus as `org.kde.StatusNotifierItem-<pid>-1`, then calls `org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem`. No GTK, no C. Left-click calls `onActivate` which calls `wailsruntime.WindowShow`.
- **Wails window** (`cmd/antaran-tray/main.go`): owns the GTK main loop. `Linux.WebviewGpuPolicy` is set to `WebviewGpuPolicyNever` — required on Nvidia proprietary drivers where hardware-accelerated WebKit silently fails to load the `wails://` URI scheme, leaving the window blank.

These two must never share a GTK main loop. `tray/daemon.go` runs in a goroutine; Wails runs on the main goroutine.

### Scanning

`tray/daemon.go` owns a `time.Ticker` at `cfg.ScanInterval` (default 30s). Each tick calls `scanner.New(root, depth).Scan(ctx)` and `process.Scan(watchList)`, then calls `app.UpdateData()` which stores results under a `sync.RWMutex`. The Wails frontend calls `app.GetScanResult()` or `app.RefreshNow()` via JS bindings.

Git commands run with a 5-second `context.WithTimeout` per repo via `os/exec`. A hung `git status` must not block the scan.

### Port detection

`internal/process/ports.go` reads `/proc/net/tcp` and `/proc/net/tcp6`, filtering for state `0A` (LISTEN). Port bytes are big-endian hex in the last 4 chars of the `local_address` field. Socket inodes are correlated to PIDs by reading `/proc/<pid>/fd/` symlinks. Do not reimplement this logic elsewhere.

### Kill audit log

`internal/process/kill.go` writes to `~/.local/share/antaran/operations.log` before sending any signal. Log entry is written first, then SIGTERM, then a 2-second wait, then SIGKILL if the process is still in `/proc/<pid>`.

### Wails JS bindings

All JS-callable methods live on `tray.App` (`tray/app.go`). Wails generates bindings at build time into `cmd/antaran-tray/frontend/wailsjs/` (gitignored). The frontend accesses them as `window.go.tray.App.<MethodName>()`. Do not add bound methods to any other type.

## Code Conventions

- `gofmt` and `go vet` are enforced by CI — both must pass before committing
- Error strings: lowercase, no trailing punctuation
- All `/proc/<pid>/` reads must handle `ENOENT` gracefully — processes vanish mid-scan
- No `fmt.Println` in non-main packages
- Tests live next to source (`foo_test.go`, same package)

## Key Invariants

- Config is optional. If `~/.config/antaran/antaran.toml` is absent, defaults apply: `scan_root=~/Coding`, `scan_interval=30s`, `git.max_depth=3`, `git.stale_after_days=14`.
- `git.fetch_remote` is `false` by default. When true, it is the only network call Antaran makes and can be slow on large repos.
- The scanner goroutine must never block the Wails UI thread.

## Build Notes (Fedora / Nvidia)

Fedora 40+ ships `webkit2gtk-4.1` but Wails hardcodes `webkit2gtk-4.0` in pkg-config. Run `make pkgconfig-shim` once to create `~/.cache/antaran-pkgconfig/webkit2gtk-4.0.pc` and export `PKG_CONFIG_PATH`.

On Nvidia proprietary drivers, `GDK_BACKEND=x11 DISPLAY=:0` is required to launch the tray app from a terminal. `make run-tray` sets these automatically. For autostart, add to `~/.config/hypr/hyprland.conf`:

```ini
exec-once = GDK_BACKEND=x11 DISPLAY=:0 antaran-tray
```

## Release

Tag format: `v0.x.x` (lowercase v). Pushing a `v*` tag triggers `.github/workflows/release.yml`, which builds for `linux/amd64` and `linux/arm64`, produces `.tar.gz` + checksums, and creates a GitHub Release.
