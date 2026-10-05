package main

import (
	"context"
	"errors"
	"time"

	"github.com/senorMk/go-peek/apps/desktop/internal/capture"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type CaptureEvent struct {
	Kind  string         `json:"kind"`
	Image *capture.Image `json:"image,omitempty"`
	Error string         `json:"error,omitempty"`
}

// CaptureScreenshot shares the same path for buttons and the native global shortcut.
// Pixels live in the preview only and are never added to provider requests.
func (d *Desktop) CaptureScreenshot(mode string) error {
	<-d.ready
	if mode != "screen" && mode != "region" {
		return errors.New("Choose screen or region capture")
	}
	if err := capture.EnsurePermission(); err != nil {
		return err
	}
	finish, ok := d.window.BeginCapture()
	if !ok {
		return errors.New("A screenshot is already in progress")
	}
	runtime.EventsEmit(d.ctx, "capture:result", CaptureEvent{Kind: "started"})
	ctx, cancel := context.WithTimeout(d.captureContext, 2*time.Minute)
	go func() {
		defer cancel()
		defer finish()
		// Allow the queued Cocoa hide and display compositor to settle before capture.
		select {
		case <-ctx.Done():
			return
		case <-time.After(250 * time.Millisecond):
		}
		image, err := capture.Take(ctx, mode)
		if d.captureContext.Err() != nil {
			return
		}
		event := CaptureEvent{Kind: "done", Image: &image}
		if errors.Is(err, capture.ErrCancelled) {
			event = CaptureEvent{Kind: "cancelled"}
		} else if err != nil {
			event = CaptureEvent{Kind: "error", Error: err.Error()}
		}
		runtime.EventsEmit(d.ctx, "capture:result", event)
	}()
	return nil
}
