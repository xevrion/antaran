package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"

	"github.com/xevrion/antaran/internal/config"
	anttray "github.com/xevrion/antaran/tray"
)

//go:embed frontend/dist
var assets embed.FS

var version = "dev"

func main() {
	var (
		cfgPath  = flag.String("config", config.DefaultPath(), "path to antaran.toml")
		showVer  = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("antaran-tray", version)
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	app := anttray.NewApp(cfg)

	// wailsApp controls the popup window. We need to be able to show/hide
	// it from the systray goroutine, so we use a channel to signal it.
	showCh := make(chan struct{}, 1)

	openWindow := func() {
		select {
		case showCh <- struct{}{}:
		default:
		}
	}

	daemon := anttray.NewDaemon(app, cfg, openWindow)

	// Run systray in its own goroutine — it needs its own OS thread on Linux.
	go daemon.Run()

	// Drain showCh and bring window to front. Wails doesn't expose a
	// Show() from outside, so we start the window hidden and toggle visibility
	// via JS window.show() called through a custom runtime bridge.
	go func() {
		for range showCh {
			// Window is toggled by the frontend polling /api/show.
			// This goroutine exists as the hook point for a future
			// wails runtime.WindowShow() call once Wails exposes it.
		}
	}()

	err = wails.Run(&options.App{
		Title:            "Antaran",
		Width:            480,
		Height:           620,
		MinWidth:         360,
		MinHeight:        400,
		DisableResize:    false,
		Frameless:        false,
		StartHidden:      false,
		HideWindowOnClose: true, // closing hides, not quits — tray keeps running
		BackgroundColour: &options.RGBA{R: 17, G: 17, B: 27, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {},
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			Icon:                []byte{}, // set via tray icon
			WindowIsTranslucent: false,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "wails error: %v\n", err)
		os.Exit(1)
	}
}
