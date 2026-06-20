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
	app      *App
	cfg      *config.Config
	openFn   func()
	rescanCh chan struct{}
}

func NewDaemon(app *App, cfg *config.Config, openFn func()) *Daemon {
	return &Daemon{
		app:      app,
		cfg:      cfg,
		openFn:   openFn,
		rescanCh: make(chan struct{}, 1),
	}
}

// TriggerRescan requests an immediate out-of-band scan (non-blocking).
// The daemon reads cfg from app so it always uses the latest settings.
func (d *Daemon) TriggerRescan() {
	select {
	case d.rescanCh <- struct{}{}:
	default:
	}
}

// Run registers the SNI icon, does an initial scan, then ticks on ScanInterval.
// Call this in a goroutine — it blocks until ctx is cancelled.
func (d *Daemon) Run(ctx context.Context) {
	icon, err := NewSNIItem(d.openFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "antaran-tray: tray icon unavailable: %v\n", err)
	} else {
		defer icon.Close()
	}

	updateIcon := func(summary string) {
		if icon != nil {
			icon.SetTooltip("antaran -- " + summary)
			icon.SetTitle("antaran")
		}
	}

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
		case <-d.rescanCh:
			// Config changed — reset ticker to new interval and scan now.
			d.app.mu.RLock()
			interval := d.app.cfg.ScanInterval
			d.app.mu.RUnlock()
			ticker.Reset(interval)
			d.scan()
			updateIcon(d.app.Summary())
		}
	}
}

// scan reads current config from app (under its lock) so it always uses
// the latest scan_root and max_depth even after a settings change.
func (d *Daemon) scan() {
	d.app.mu.RLock()
	root := d.app.cfg.ScanRoot
	depth := d.app.cfg.Git.MaxDepth
	watch := d.app.cfg.Process.Watch
	d.app.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sc := scanner.New(root, depth)
	repos, _ := sc.Scan(ctx)
	procs, _ := process.Scan(watch)
	d.app.UpdateData(repos, procs)
}
