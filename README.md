<h1 align="center">Antaran</h1>
<h3 align="center">अंतरण &mdash; knows what your dev folder is hiding</h3>

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
  A native Linux system tray daemon that watches your dev environment and surfaces what is actually consuming your machine and attention.
</p>

## What it does

You open a new terminal, start a dev server, fix a bug in three repos, and then forget about all of it. Two weeks later your machine is sluggish and you have 11 git repos with uncommitted changes you don't remember touching.

Antaran watches your `~/Coding` folder and tells you:

- **Dirty repos** — which git repos have uncommitted changes, unpushed commits, or branches that haven't been touched in weeks
- **Zombie dev servers** — which node/cargo/vite/bun processes are still running, what ports they're listening on, how long they've been up, and how much RAM they're eating
- **One-click actions** (tray UI) — kill a process, open a repo in your editor

The tray icon shows a live summary:

```
38 dirty repos · 3 zombie dev servers eating 94MB
```

Click to expand the full list.

## Features

- Scans a configurable root folder (default `~/Coding`) for git repos
- Detects uncommitted changes, unpushed commits, and stale branches
- Finds long-running dev processes via `/proc` directly -- no `ps`, no shell subprocesses
- Shows which TCP ports each process is listening on
- System tray icon via StatusNotifierItem (works with Hyprland, KDE, GNOME)
- Click to expand a dark-themed popup window with repo and process details
- Kill button with SIGTERM then SIGKILL escalation and audit log
- Also ships as a standalone CLI (`antaran`) for scripting and CI use
- Hyprland-first, graceful on other Wayland and X11 setups

## Install

### Tray app from source

Requires Go >= 1.21, Wails v2, and `libwebkit2gtk`.

```bash
git clone https://github.com/xevrion/antaran.git
cd antaran

# Fedora 40+ only: create a webkit2gtk-4.0 shim once
make pkgconfig-shim
export PKG_CONFIG_PATH="$HOME/.cache/antaran-pkgconfig:$PKG_CONFIG_PATH"

make build-tray
make install-tray    # installs to ~/.local/bin/antaran-tray
```

Add to your Hyprland config:

```ini
exec-once = GDK_BACKEND=x11 DISPLAY=:0 antaran-tray
```

### CLI only (no Wails required)

```bash
make build
make install    # installs to ~/.local/bin/antaran
```

Or from a release tarball:

```bash
curl -Lo antaran.tar.gz \
  https://github.com/xevrion/antaran/releases/latest/download/antaran-linux-amd64.tar.gz
tar -xzf antaran.tar.gz
install -Dm755 antaran-linux-amd64 ~/.local/bin/antaran
```

## CLI usage

```bash
antaran                          # scan ~/Coding
antaran --root ~/projects        # override scan root
antaran --json                   # JSON output for scripting
antaran --config /path/to/antaran.toml
```

### Example output

```
antaran -- scanning /home/yash/Coding

  38 dirty repos · 3 zombie dev servers eating 94MB

-- git repos (114 scanned) --
  antaran                         [main]  2 staged, 1 untracked
  marknote                        [feat/block-editor]  5 unstaged
  nyamp                           [main]  stale 27d

-- dev processes --
  node      pid:6173    up:10h24m    7MB    :3000
  bun       pid:2782823  up:24m      74MB   :8080
```

## Configuration

```bash
mkdir -p ~/.config/antaran
cp antaran.toml.example ~/.config/antaran/antaran.toml
```

Config is optional -- Antaran runs with sensible defaults if the file is absent.

| Key | Default | Description |
|-----|---------|-------------|
| `scan_root` | `~/Coding` | Root directory to scan for git repos |
| `scan_interval` | `30s` | How often to refresh in tray mode |
| `git.max_depth` | `3` | Directory depth to search for repos |
| `git.stale_after_days` | `14` | Days of inactivity before a branch is flagged |
| `git.fetch_remote` | `false` | Fetch before checking for unpushed commits |
| `process.watch` | `[node, cargo, vite, ...]` | Process names to watch |

See [`antaran.toml.example`](antaran.toml.example) for the full reference.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Adding a new watcher is intentionally small -- implement one interface, write a test.

## License

MIT -- see [LICENSE](LICENSE).
