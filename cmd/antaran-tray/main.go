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
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/xevrion/antaran/internal/config"
	anttray "github.com/xevrion/antaran/tray"
)

//go:embed frontend/dist
var assets embed.FS

var version = "dev"

func main() {
	var (
		cfgPath = flag.String("config", config.DefaultPath(), "path to antaran.toml")
		showVer = flag.Bool("version", false, "print version and exit")
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

	// wailsCtx is populated by OnStartup — used to show/hide the window.
	var wailsCtx context.Context
	ctxReady := make(chan struct{})

	openWindow := func() {
		// Wait until Wails has started before trying to show the window
		<-ctxReady
		wailsruntime.WindowShow(wailsCtx)
		wailsruntime.WindowSetAlwaysOnTop(wailsCtx, true)
		wailsruntime.WindowSetAlwaysOnTop(wailsCtx, false)
	}

	// cancelDaemon is called when Wails shuts down
	daemonCtx, cancelDaemon := context.WithCancel(context.Background())

	daemon := anttray.NewDaemon(app, cfg, openWindow)
	go daemon.Run(daemonCtx)

	err = wails.Run(&options.App{
		Title:             "Antaran",
		Width:             480,
		Height:            620,
		MinWidth:          360,
		MinHeight:         400,
		DisableResize:     false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: 17, G: 17, B: 27, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			wailsCtx = ctx
			close(ctxReady)
		},
		OnShutdown: func(ctx context.Context) {
			cancelDaemon()
		},
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			Icon:                []byte{},
			WindowIsTranslucent: false,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "wails error: %v\n", err)
		os.Exit(1)
	}
}
