# FAQ & Troubleshooting

## Build Issues

### `libwebkit2gtk` not found (Wails build only)

On Ubuntu 22.04:
```bash
sudo apt install libwebkit2gtk-4.0-dev libgtk-3-dev pkg-config
```

On Ubuntu 24.04:
```bash
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev pkg-config
```

On Fedora/Arch, the package name differs — search your distro's package index for `webkit2gtk`.

The CLI (`go build ./cmd/antaran`) does **not** require webkit2gtk. Only the Wails tray app does.

### `wails: command not found`

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Make sure `$(go env GOPATH)/bin` is in your `$PATH`.

## Runtime Issues

### Antaran shows 0 repos

Check that `scan_root` in your config points to the right directory and that it contains git repos at or within `max_depth` levels. Run with `--root /path/to/your/code` to override.

### Process watcher shows nothing

Antaran reads `/proc/<pid>/comm` and matches it against your `watch` list. The `comm` name is the short process name (max 15 chars, no path). For example, a `node` process running `server.js` has `comm = node`.

To see what names your processes have:
```bash
ls /proc/$(pgrep node)/comm 2>/dev/null && cat /proc/$(pgrep node)/comm
```

### Port detection doesn't work / shows wrong ports

Port detection reads `/proc/net/tcp` (IPv4) and `/proc/net/tcp6` (IPv6) and only reports **listening** ports (state `LISTEN`). Ephemeral connection ports are intentionally ignored.

If ports aren't showing, the process may be using a Unix socket or QUIC instead of TCP.

### Permission errors on `/proc/<pid>/fd`

Antaran reads `/proc/<pid>/fd` to correlate socket inodes to processes. This requires that the process running `antaran` can read other processes' fd directories. On most Linux systems this works for the current user's own processes. Antaran silently skips any pid it can't read.

## Tray App Issues

### Tray icon doesn't appear on Hyprland

Hyprland requires a system tray implementation (e.g. `waybar` with the `tray` module enabled). If the icon doesn't appear, check that your bar has tray support running.

### App window doesn't open on click

On some Wayland compositors, the Wails window may require `XDG_RUNTIME_DIR` to be set. This is typically set automatically by your login manager or `seatd`.
