<div align="center">
  <img src="assets/icon.png" width="120" alt="Antaran icon" />
</div>

<h1 align="center">Antaran</h1>
<h3 align="center">अंतरण — knows what your dev folder is hiding</h3>

<p align="center">
  <a href="https://github.com/xevrion/antaran/actions/workflows/ci.yml">
    <img alt="CI" src="https://img.shields.io/github/actions/workflow/status/xevrion/antaran/ci.yml?branch=main&style=flat-square&label=CI" />
  </a>
  <a href="https://github.com/xevrion/antaran/releases">
    <img alt="GitHub release" src="https://img.shields.io/github/v/release/xevrion/antaran?style=flat-square" />
  </a>
  <a href="LICENSE">
    <img alt="License" src="https://img.shields.io/github/license/xevrion/antaran?style=flat-square" />
  </a>
  <a href="https://github.com/xevrion/antaran/stargazers">
    <img alt="Stars" src="https://img.shields.io/github/stars/xevrion/antaran?style=flat-square" />
  </a>
</p>

<p align="center">
  A native Linux system tray daemon that watches your dev environment and surfaces what's actually consuming your machine and attention.
</p>

---

## What it does

You open a new terminal, start a dev server, fix a bug in three repos, and then forget about all of it. Two weeks later your machine is sluggish and you have 11 git repos with uncommitted changes you don't remember touching.

Antaran watches your `~/Coding` folder and tells you:

- **Dirty repos** — which git repos have uncommitted changes, unpushed commits, or branches that haven't been touched in weeks
- **Zombie dev servers** — which node/cargo/vite/bun processes are still running, what ports they're listening on, how long they've been up, and how much RAM they're eating
- **One-click actions** (tray UI) — kill a process, open a repo in your editor, copy git status to clipboard

The tray icon shows a live summary:

```
3 dirty repos · 2 zombie dev servers eating 118MB
```

Click it to expand the full list.

## Features

- Scans a configurable root folder (default `~/Coding`) for git repos
- Detects uncommitted changes, unpushed commits, and stale branches
- Finds long-running dev processes via `/proc` — no `ps`, no shell subprocesses
- Shows which ports each process is listening on (reads `/proc/net/tcp` directly)
- Works as a CLI tool today, tray daemon coming next
- Single static binary, ~10MB, zero runtime dependencies
- Hyprland-first, graceful on other Wayland/X11 setups

## Install

### From release (recommended)

```bash
curl -Lo antaran.tar.gz \
  https://github.com/xevrion/antaran/releases/latest/download/antaran-linux-amd64.tar.gz
tar -xzf antaran.tar.gz
install -Dm755 antaran-linux-amd64 ~/.local/bin/antaran
```

### From source

```bash
git clone https://github.com/xevrion/antaran.git
cd antaran
make build
make install   # installs to ~/.local/bin/antaran
```

Requires Go ≥ 1.21. No other build dependencies for the CLI.

## Usage

```bash
# Scan ~/Coding (or whatever scan_root is set to)
antaran

# Override the scan root
antaran --root ~/projects

# JSON output (for scripting or tray integration)
antaran --json

# Use a custom config file
antaran --config /path/to/antaran.toml
```

### Example output

```
antaran — scanning /home/yash/Coding

  3 dirty repos · 2 zombie dev servers eating 118MB

── git repos (42 scanned) ──
  antaran                         [main]  2 staged, 1 untracked
  marknote                        [feat/block-editor]  5 unstaged
  nyamp                           [main]  stale 27d

── dev processes ──
  node      pid:6173    up:10h24m    7MB    :3000
  bun       pid:2782823  up:24m      74MB   :8080
```

## Configuration

Copy the example config and edit:

```bash
mkdir -p ~/.config/antaran
cp antaran.toml.example ~/.config/antaran/antaran.toml
```

Key options:

| Key | Default | Description |
|-----|---------|-------------|
| `scan_root` | `~/Coding` | Root directory to scan for git repos |
| `scan_interval` | `30s` | How often to refresh (tray mode) |
| `git.max_depth` | `3` | How deep to search for repos |
| `git.stale_after_days` | `14` | Days before a quiet branch is "stale" |
| `git.fetch_remote` | `false` | Fetch before checking unpushed commits |
| `process.watch` | `[node, cargo, vite, ...]` | Process names to watch |

See [`antaran.toml.example`](antaran.toml.example) for the full reference.

## Tray app (coming soon)

The CLI is the working core. The Wails tray UI is next — it will wrap the same scanning logic in a native popup window with one-click actions. Requires `libwebkit2gtk`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Adding a new watcher is intentionally small — implement one interface, register it, write a test.

## License

MIT — see [LICENSE](LICENSE).

---

<p align="center">
  Built by <a href="https://github.com/xevrion">Yash Bavadiya</a> · IIT Jodhpur · GSoC 2026 prep
</p>
