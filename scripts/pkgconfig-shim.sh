#!/usr/bin/env bash
# On Fedora 40+, webkit2gtk ships as -4.1, but Wails v2 looks for -4.0.
# This creates a local shim so builds work without patching Wails.
# Usage: source scripts/pkgconfig-shim.sh (or eval it in your shell profile)
SHIM_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/antaran-pkgconfig"
mkdir -p "$SHIM_DIR"
if [ ! -f "$SHIM_DIR/webkit2gtk-4.0.pc" ]; then
  VER=$(pkg-config --modversion webkit2gtk-4.1 2>/dev/null || echo "2.44.0")
  cat > "$SHIM_DIR/webkit2gtk-4.0.pc" <<PCEOF
Name: webkit2gtk-4.0
Description: WebKit2 Gtk+ (4.0 shim -> 4.1)
Version: $VER
Requires: webkit2gtk-4.1
Libs:
Cflags:
PCEOF
fi
export PKG_CONFIG_PATH="$SHIM_DIR:${PKG_CONFIG_PATH}"
echo "PKG_CONFIG_PATH set to: $PKG_CONFIG_PATH"
