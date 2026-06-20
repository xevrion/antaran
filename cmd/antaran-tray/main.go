package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
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
var assetsFS embed.FS

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

	assets, err := fs.Sub(assetsFS, "frontend/dist")
	if err != nil {
		fmt.Fprintf(os.Stderr, "assets error: %v\n", err)
		os.Exit(1)
	}

	app := anttray.NewApp(cfg, *cfgPath)

	var wailsCtx context.Context
	ctxReady := make(chan struct{})

	openWindow := func() {
		<-ctxReady
		wailsruntime.WindowShow(wailsCtx)
		wailsruntime.WindowSetAlwaysOnTop(wailsCtx, true)
		wailsruntime.WindowSetAlwaysOnTop(wailsCtx, false)
	}

	daemonCtx, cancelDaemon := context.WithCancel(context.Background())
	daemon := anttray.NewDaemon(app, cfg, openWindow)

	// When config changes, tell the daemon to rescan immediately with new settings.
	app.SetOnConfigChange(func() {
		daemon.TriggerRescan()
	})

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
			app.SetContext(ctx)
			close(ctxReady)
		},
		OnShutdown: func(ctx context.Context) {
			cancelDaemon()
		},
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			Icon:             []byte{},
			WebviewGpuPolicy: linux.WebviewGpuPolicyNever,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "wails error: %v\n", err)
		os.Exit(1)
	}
}
