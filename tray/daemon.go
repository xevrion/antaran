package tray

import (
	"context"
	"time"

	"github.com/getlantern/systray"
	"github.com/xevrion/antaran/internal/config"
	"github.com/xevrion/antaran/internal/process"
	"github.com/xevrion/antaran/internal/scanner"
)

// Daemon runs the tray icon loop. It owns the scan ticker and updates
// the App state so the Wails window always has fresh data when opened.
type Daemon struct {
	app    *App
	cfg    *config.Config
	openFn func() // called to bring the Wails window to front
}

func NewDaemon(app *App, cfg *config.Config, openFn func()) *Daemon {
	return &Daemon{app: app, cfg: cfg, openFn: openFn}
}

// Run starts the systray. This call blocks until the tray icon is removed.
func (d *Daemon) Run() {
	systray.Run(d.onReady, d.onExit)
}

func (d *Daemon) onReady() {
	systray.SetTitle("antaran")
	systray.SetTooltip("antaran — scanning...")
	systray.SetIcon(iconPNG())

	mShow := systray.AddMenuItem("Show", "Open the Antaran window")
	systray.AddSeparator()
	mRefresh := systray.AddMenuItem("Refresh now", "Rescan immediately")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Stop Antaran")

	// Initial scan
	d.scan()

	// Periodic scan ticker
	ticker := time.NewTicker(d.cfg.ScanInterval)

	go func() {
		for {
			select {
			case <-ticker.C:
				d.scan()
			case <-mShow.ClickedCh:
				d.openFn()
			case <-mRefresh.ClickedCh:
				d.scan()
			case <-mQuit.ClickedCh:
				ticker.Stop()
				systray.Quit()
				return
			}
		}
	}()
}

func (d *Daemon) onExit() {}

func (d *Daemon) scan() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sc := scanner.New(d.cfg.ScanRoot, d.cfg.Git.MaxDepth)
	repos, _ := sc.Scan(ctx)
	procs, _ := process.Scan(d.cfg.Process.Watch)

	d.app.UpdateData(repos, procs)

	summary := d.app.Summary()
	systray.SetTooltip("antaran — " + summary)
	systray.SetTitle("⚙ " + summary)
}
