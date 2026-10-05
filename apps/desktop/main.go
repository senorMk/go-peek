package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/senorMk/go-peek/apps/desktop/internal/credentials"
	"github.com/senorMk/go-peek/apps/desktop/internal/overlay"
	"github.com/senorMk/go-peek/apps/desktop/internal/platform"
	"github.com/senorMk/go-peek/apps/desktop/internal/session"
	"github.com/senorMk/go-peek/apps/desktop/internal/settings"
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
	ready                     chan struct{}
	requestMu                 sync.Mutex
	assistant                 *session.Assistant
	config                    settings.Config
	configPath                string
	settingsError             string
	ctx                       context.Context
	streams                   *session.Manager
	mu                        sync.Mutex
	window                    *overlay.Controller
	stopShortcut              func()
	stopCaptureShortcut       func()
	captureContext            context.Context
	captureShortcutRegistered bool
	captureShortcutError      string
	stopLifecycle             context.CancelFunc
	shortcutReady             sync.Once
	alwaysOnTop               bool
	shortcutRegistered        bool
	shortcutError             string
	protection                bool
}

type Status struct {
	AlwaysOnTop                bool   `json:"alwaysOnTop"`
	CaptureProtectionRequested bool   `json:"captureProtectionRequested"`
	Shortcut                   string `json:"shortcut"`
	ShortcutRegistered         bool   `json:"shortcutRegistered"`
	ShortcutError              string `json:"shortcutError,omitempty"`
	CaptureShortcut            string `json:"captureShortcut"`
	CaptureShortcutRegistered  bool   `json:"captureShortcutRegistered"`
	CaptureShortcutError       string `json:"captureShortcutError,omitempty"`
}

func (d *Desktop) GetStatus() Status {
	d.mu.Lock()
	defer d.mu.Unlock()
	return Status{AlwaysOnTop: d.alwaysOnTop, CaptureProtectionRequested: d.protection, Shortcut: platform.ShortcutLabel, ShortcutRegistered: d.shortcutRegistered, ShortcutError: d.shortcutError, CaptureShortcut: platform.CaptureShortcutLabel, CaptureShortcutRegistered: d.captureShortcutRegistered, CaptureShortcutError: d.captureShortcutError}
}

func (d *Desktop) StartDemo(marker string) (string, error) {
	<-d.ready
	return d.streams.Start(d.ctx, marker)
}
func (d *Desktop) Cancel() {
	<-d.ready
	d.assistant.Cancel()
}
func (d *Desktop) Reset() {
	<-d.ready
	d.assistant.Reset()
}
func (d *Desktop) CancelDemo() { <-d.ready; d.streams.Cancel() }
func (d *Desktop) ResetDemo()  { <-d.ready; d.streams.Reset() }

func (d *Desktop) SetAlwaysOnTop(enabled bool) error {
	<-d.ready
	d.requestMu.Lock()
	defer d.requestMu.Unlock()
	if d.configPath == "" {
		return fmt.Errorf("local settings directory is unavailable")
	}
	cfg := d.config
	cfg.AlwaysOnTop = enabled
	if err := settings.Save(d.configPath, cfg); err != nil {
		return err
	}
	d.config, d.settingsError = cfg, ""
	d.mu.Lock()
	runtime.WindowSetAlwaysOnTop(d.ctx, enabled)
	d.alwaysOnTop = enabled
	d.mu.Unlock()
	runtime.EventsEmit(d.ctx, "prototype:status", d.GetStatus())
	return nil
}

// Timed restore remains available when the system shortcut is unavailable.
func (d *Desktop) HideTemporarily() {
	<-d.ready
	d.window.HideFor(3 * time.Second)
}

func (d *Desktop) registerShortcut(ctx context.Context) {
	<-d.ready
	d.shortcutReady.Do(func() {
		captureEvents, captureCleanup, captureErr := platform.RegisterCaptureShortcut()
		d.mu.Lock()
		if captureErr != nil {
			d.captureShortcutError = captureErr.Error()
		} else {
			d.captureShortcutRegistered = true
			d.stopCaptureShortcut = captureCleanup
		}
		d.mu.Unlock()
		if captureErr == nil {
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-captureEvents:
						_ = d.CaptureScreenshot("screen")
					}
				}
			}()
		}
		events, cleanup, err := platform.RegisterShortcut()
		d.mu.Lock()
		if err != nil {
			d.shortcutError = err.Error()
		} else {
			d.shortcutRegistered = true
			d.stopShortcut = cleanup
		}
		d.mu.Unlock()
		runtime.EventsEmit(d.ctx, "prototype:status", d.GetStatus())
		if err == nil {
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-events:
						d.window.ToggleWithRestore(func() {
							platform.MoveWindowToCurrentDisplay()
							runtime.WindowShow(d.ctx)
						})
					}
				}
			}()
		}
	})
}

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "--store-key" {
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "Usage: --store-key openai|zen (key supplied on stdin)")
			os.Exit(2)
		}
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 4097))
		if err == nil && len(data) > 4096 {
			err = fmt.Errorf("API key input exceeds 4096 bytes")
		}
		if err == nil {
			err = credentials.Set(os.Args[2], string(data))
		}
		for i := range data {
			data[i] = 0
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("API key saved to macOS Keychain. Refresh provider settings in GoPeek.")
		return
	}
	// Run both variants against the same recorder to establish a control.
	protection := os.Getenv("GOPEEK_CAPTURE_PROTECTION") != "off"
	desktop := &Desktop{protection: protection, ready: make(chan struct{})}
	lifecycle, cancelLifecycle := context.WithCancel(context.Background())
	desktop.stopLifecycle = cancelLifecycle
	desktop.captureContext = lifecycle
	desktop.config = settings.Default()
	path, err := settings.Path()
	if err != nil {
		desktop.settingsError = err.Error()
	} else {
		desktop.configPath = path
		cfg, err := settings.Load(path)
		if err != nil {
			desktop.settingsError = err.Error()
		} else {
			desktop.config = cfg
		}
	}

	desktop.alwaysOnTop = desktop.config.AlwaysOnTop
	err = wails.Run(&options.App{
		Title:       "GoPeek",
		AlwaysOnTop: desktop.config.AlwaysOnTop,
		Width:       520, Height: 620, MinWidth: 420, MinHeight: 360,
		BackgroundColour: options.NewRGB(18, 22, 29),
		AssetServer:      &assetserver.Options{Assets: assets},
		Mac:              &mac.Options{ContentProtection: protection, TitleBar: mac.TitleBarHidden(), Appearance: mac.NSAppearanceNameDarkAqua},
		OnStartup: func(ctx context.Context) {
			desktop.ctx = ctx
			desktop.assistant = session.NewAssistant(func(event session.Event) { runtime.EventsEmit(ctx, "assistant:stream", event) })
			desktop.window = overlay.New(func() { runtime.WindowShow(ctx) }, func() { runtime.WindowHide(ctx) })
			desktop.streams = session.New(func(event session.Event) { runtime.EventsEmit(ctx, "demo:stream", event) })
			close(desktop.ready)
		},
		OnDomReady: func(_ context.Context) { desktop.registerShortcut(lifecycle) },
		OnShutdown: func(_ context.Context) {
			<-desktop.ready
			desktop.stopLifecycle()
			desktop.window.Close()
			desktop.streams.Reset()
			desktop.assistant.Reset()
			desktop.mu.Lock()
			defer desktop.mu.Unlock()
			if desktop.stopCaptureShortcut != nil {
				desktop.stopCaptureShortcut()
			}
			if desktop.stopShortcut != nil {
				desktop.stopShortcut()
			}
		},
		Bind: []interface{}{desktop},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Desktop prototype failed to start.")
		os.Exit(1)
	}
}
