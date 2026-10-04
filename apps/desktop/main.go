package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/senorMk/assessment-caddy/apps/desktop/internal/session"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// Desktop is the narrow frontend bridge. Lifecycle functions stay outside the bindings.
type Desktop struct {
	ctx        context.Context
	streams    *session.Manager
	mu         sync.Mutex
	restore    context.CancelFunc
	protection bool
}

type Status struct {
	CaptureProtectionRequested bool `json:"captureProtectionRequested"`
}

func (d *Desktop) GetStatus() Status { return Status{CaptureProtectionRequested: d.protection} }

func (d *Desktop) StartDemo(marker string) (string, error) { return d.streams.Start(d.ctx, marker) }
func (d *Desktop) Cancel()                                 { d.streams.Cancel() }
func (d *Desktop) Reset()                                  { d.streams.Reset() }
func (d *Desktop) SetAlwaysOnTop(enabled bool)             { runtime.WindowSetAlwaysOnTop(d.ctx, enabled) }

// Timed restore lets a user test hiding without a global shortcut or tray dependency.
func (d *Desktop) HideTemporarily() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.restore != nil {
		d.restore()
	}
	ctx, cancel := context.WithCancel(d.ctx)
	d.restore = cancel
	runtime.WindowHide(d.ctx)
	go func() {
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			runtime.WindowShow(d.ctx)
		}
	}()
}

func main() {
	// Run both variants against the same recorder to establish a control.
	protection := os.Getenv("CADDY_CAPTURE_PROTECTION") != "off"
	desktop := &Desktop{protection: protection}
	err := wails.Run(&options.App{
		Title: "Assessment Caddy — capture feasibility",
		Width: 520, Height: 700, MinWidth: 400, MinHeight: 520,
		BackgroundColour: options.NewRGB(18, 22, 29),
		AssetServer:      &assetserver.Options{Assets: assets},
		Mac:              &mac.Options{ContentProtection: protection},
		OnStartup: func(ctx context.Context) {
			desktop.ctx = ctx
			desktop.streams = session.New(func(event session.Event) { runtime.EventsEmit(ctx, "demo:stream", event) })
		},
		OnShutdown: func(_ context.Context) {
			desktop.streams.Reset()
			desktop.mu.Lock()
			defer desktop.mu.Unlock()
			if desktop.restore != nil {
				desktop.restore()
			}
		},
		Bind: []interface{}{desktop},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Desktop prototype failed to start.")
		os.Exit(1)
	}
}
