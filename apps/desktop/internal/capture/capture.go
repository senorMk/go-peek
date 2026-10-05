// Package capture performs explicit, local-only screenshots without logging pixels.
package capture

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type Image struct {
	DataURL string `json:"dataUrl"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

var ErrCancelled = errors.New("Screenshot cancelled")

const maxBytes = 20 * 1024 * 1024

type Runner func(context.Context, string, ...string) error

func Take(ctx context.Context, mode string) (Image, error) {
	if runtime.GOOS != "darwin" {
		return Image{}, errors.New("Screenshot capture requires macOS")
	}
	return take(ctx, mode, func(ctx context.Context, command string, args ...string) error {
		return exec.CommandContext(ctx, command, args...).Run()
	}, selectRegion)
}

func take(ctx context.Context, mode string, run Runner, selectRect func(context.Context) (Rect, error)) (Image, error) {
	if mode != "screen" && mode != "region" {
		return Image{}, errors.New("Choose screen or region capture")
	}
	var region string
	if mode == "region" {
		rect, err := selectRect(ctx)
		if err != nil {
			return Image{}, err
		}
		region, err = rect.argument()
		if err != nil {
			return Image{}, err
		}
		// Native selector panels are closed before capturing; wait for compositing.
		select {
		case <-ctx.Done():
			return Image{}, ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	dir, err := os.MkdirTemp("", "gopeek-capture-")
	if err != nil {
		return Image{}, errors.New("Could not prepare screenshot")
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "capture.png")
	args := []string{"-x", "-t", "png"}
	if mode == "region" {
		args = append(args, "-R", region)
	} else {
		args = append(args, "-D", "1")
	}
	args = append(args, path)
	err = run(ctx, "/usr/sbin/screencapture", args...)
	if ctx.Err() != nil {
		return Image{}, errors.New("Screenshot stopped or timed out")
	}
	if err != nil {
		return Image{}, commandError(err)
	}
	file, err := os.Open(path)
	if err != nil {
		return Image{}, errors.New("Could not read screenshot")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return Image{}, errors.New("Could not read screenshot")
	}
	if len(data) > maxBytes {
		return Image{}, errors.New("Screenshot exceeds 20 MiB. Capture a smaller region")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 64_000_000 {
		return Image{}, errors.New("Screenshot is invalid or exceeds 64 megapixels. Capture a smaller region")
	}
	return Image{DataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), Width: cfg.Width, Height: cfg.Height}, nil
}

// commandError exposes only an exit status or a fixed category, never stderr,
// which can include file paths or other private details.
func commandError(err error) error {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return fmt.Errorf("macOS screenshot command failed (exit %d). This is not a confirmed permission denial. Ensure the desktop is unlocked; if GoPeek was rebuilt, re-add the current .app in Screen & System Audio Recording and reopen it", exit.ExitCode())
	}
	if errors.Is(err, os.ErrPermission) {
		return errors.New("macOS blocked launching the screenshot utility; this is separate from screen recording access")
	}
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("The macOS screenshot utility was not found")
	}
	return errors.New("Could not run the macOS screenshot utility")
}
