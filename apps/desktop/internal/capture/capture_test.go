package capture

import (
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureUsesNativeArgumentsAndRemovesPrivateFile(t *testing.T) {
	for _, mode := range []string{"screen", "region"} {
		t.Run(mode, func(t *testing.T) {
			var path string
			got, err := take(context.Background(), mode, func(ctx context.Context, command string, args ...string) error {
				if command != "/usr/sbin/screencapture" {
					t.Fatal(command)
				}
				if mode == "screen" && (args[3] != "-D" || args[4] != "1") {
					t.Fatal(args)
				}
				if mode == "region" && (args[3] != "-R" || args[4] != "-10,20,100,80") {
					t.Fatal(args)
				}
				path = args[len(args)-1]
				stat, err := os.Stat(filepath.Dir(path))
				if err != nil {
					t.Fatal(err)
				}
				if stat.Mode().Perm() != 0700 {
					t.Fatal("capture directory must be private")
				}
				f, err := os.Create(path)
				if err != nil {
					return err
				}
				defer f.Close()
				return png.Encode(f, image.NewRGBA(image.Rect(0, 0, 2, 3)))
			}, fixtureRegion)
			if err != nil || got.Width != 2 || got.Height != 3 || got.DataURL == "" {
				t.Fatalf("%+v %v", got, err)
			}
			if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
				t.Fatal("temporary pixels remain")
			}
		})
	}
}

func TestCancelledAndFailedCaptureDoNotReturnPixels(t *testing.T) {
	got, err := take(context.Background(), "region", func(context.Context, string, ...string) error { return nil }, func(context.Context) (Rect, error) { return Rect{}, ErrCancelled })
	if !errors.Is(err, ErrCancelled) || got.DataURL != "" {
		t.Fatalf("%+v %v", got, err)
	}
	_, err = take(context.Background(), "screen", func(context.Context, string, ...string) error { return errors.New("private screen title") }, fixtureRegion)
	if err == nil || err.Error() == "private screen title" {
		t.Fatal("raw process error leaked")
	}
	_, err = take(context.Background(), "screen", func(ctx context.Context, command string, args ...string) error {
		return os.WriteFile(args[len(args)-1], []byte("not a PNG"), 0600)
	}, fixtureRegion)
	if err == nil {
		t.Fatal("invalid screenshot accepted")
	}
}

func TestCommandFailuresAreNotAllReportedAsScreenPermissionDenials(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{os.ErrPermission, "macOS blocked launching the screenshot utility; this is separate from screen recording access"},
		{os.ErrNotExist, "The macOS screenshot utility was not found"},
		{errors.New("private screen title and file path"), "Could not run the macOS screenshot utility"},
	}
	for _, tt := range cases {
		if got := commandError(tt.err).Error(); got != tt.want {
			t.Fatalf("got %q, want %q", got, tt.want)
		}
	}
}

func fixtureRegion(context.Context) (Rect, error) { return Rect{-10, 20, 100, 80}, nil }
