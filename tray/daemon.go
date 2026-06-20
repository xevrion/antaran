package tray

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/xevrion/antaran/internal/config"
	"github.com/xevrion/antaran/internal/process"
	"github.com/xevrion/antaran/internal/scanner"
)

// Daemon owns the scan ticker and the SNI tray icon.
// It runs entirely on its own goroutine — no GTK involvement.
type Daemon struct {
	app    *App
	cfg    *config.Config
	openFn func()
}

func NewDaemon(app *App, cfg *config.Config, openFn func()) *Daemon {
	return &Daemon{app: app, cfg: cfg, openFn: openFn}
}

// Run registers the SNI icon, does an initial scan, then ticks on ScanInterval.
// Call this in a goroutine — it blocks until ctx is cancelled.
func (d *Daemon) Run(ctx context.Context) {
	icon, err := NewSNIItem(d.openFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "antaran-tray: tray icon unavailable: %v\n", err)
		// Continue without tray — the Wails window still works
	} else {
		defer icon.Close()
	}

	updateIcon := func(summary string) {
		if icon != nil {
			icon.SetTooltip("antaran — " + summary)
			icon.SetTitle("antaran")
		}
	}

	// Initial scan
	d.scan()
	updateIcon(d.app.Summary())

	ticker := time.NewTicker(d.cfg.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.scan()
			updateIcon(d.app.Summary())
		}
	}
}

func (d *Daemon) scan() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sc := scanner.New(d.cfg.ScanRoot, d.cfg.Git.MaxDepth)
	repos, _ := sc.Scan(ctx)
	procs, _ := process.Scan(d.cfg.Process.Watch)
	d.app.UpdateData(repos, procs)
}
