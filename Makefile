.PHONY: build build-tray run test fmt lint clean install install-tray pkgconfig-shim

BINARY      := bin/antaran
TRAY_BINARY := bin/antaran-tray
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -ldflags="-s -w -X main.version=$(VERSION)"
TRAY_TAGS   := legacy_appindicator
WAILS       := $(shell command -v wails 2>/dev/null || echo "$(HOME)/go/bin/wails")

# On Fedora 40+, webkit2gtk ships as -4.1 but Wails looks for -4.0.
# Run `make pkgconfig-shim` once to create the shim, then export PKG_CONFIG_PATH.
SHIM_DIR := $(HOME)/.cache/antaran-pkgconfig

pkgconfig-shim:
	mkdir -p $(SHIM_DIR)
	@printf 'Name: webkit2gtk-4.0\nDescription: shim -> 4.1\nVersion: %s\nRequires: webkit2gtk-4.1\nLibs:\nCflags:\n' \
	  "$$(pkg-config --modversion webkit2gtk-4.1 2>/dev/null || echo 2.44.0)" \
	  > $(SHIM_DIR)/webkit2gtk-4.0.pc
	@echo "Shim written to $(SHIM_DIR)/webkit2gtk-4.0.pc"
	@echo "Run: export PKG_CONFIG_PATH=$(SHIM_DIR):\$$PKG_CONFIG_PATH"

build:
	go build $(LDFLAGS) -tags $(TRAY_TAGS) -o $(BINARY) ./cmd/antaran

build-tray: pkgconfig-shim
	PKG_CONFIG_PATH=$(SHIM_DIR):$(PKG_CONFIG_PATH) \
	  $(WAILS) build -tags $(TRAY_TAGS) \
	  -o ../../../$(TRAY_BINARY) \
	  -projectdir cmd/antaran-tray

run:
	go run -tags $(TRAY_TAGS) ./cmd/antaran $(ARGS)

test:
	go test -race -tags $(TRAY_TAGS) ./...

fmt:
	gofmt -w .

lint:
	go vet -tags $(TRAY_TAGS) ./...

clean:
	rm -rf bin/ dist/ cmd/antaran-tray/bin/

install: build
	install -Dm755 $(BINARY) ~/.local/bin/antaran

install-tray: build-tray
	install -Dm755 $(TRAY_BINARY) ~/.local/bin/antaran-tray
