package capture

import (
	"context"
	"errors"
	"math"
	"testing"
)

func TestRegionArgumentHandlesFractionalAndNegativeDesktopCoordinates(t *testing.T) {
	for _, tt := range []struct {
		rect Rect
		want string
	}{
		{Rect{10.5, 20.2, 100.1, 80.3}, "10,20,101,81"},
		{Rect{-1920, -1080, 800, 600}, "-1920,-1080,800,600"},
		{Rect{-5.5, 2, 2.5, 4}, "-6,2,3,4"},
	} {
		got, err := tt.rect.argument()
		if err != nil || got != tt.want {
			t.Fatalf("%+v: %q %v, want %q", tt.rect, got, err, tt.want)
		}
	}
	for _, rect := range []Rect{{0, 0, 1, 50}, {0, 0, 50, 0}, {math.NaN(), 0, 5, 5}, {0, math.Inf(1), 5, 5}} {
		if _, err := rect.argument(); err == nil {
			t.Fatalf("invalid region accepted: %+v", rect)
		}
	}
}

func TestRegionCancellationNeverInvokesScreenshotCommand(t *testing.T) {
	run := func(context.Context, string, ...string) error { t.Fatal("command ran after cancellation"); return nil }
	for _, want := range []error{ErrCancelled, context.Canceled} {
		_, err := take(context.Background(), "region", run, func(context.Context) (Rect, error) { return Rect{}, want })
		if !errors.Is(err, want) {
			t.Fatalf("got %v, want %v", err, want)
		}
	}
}

func TestInvalidSelectionNeverInvokesScreenshotCommand(t *testing.T) {
	_, err := take(context.Background(), "region", func(context.Context, string, ...string) error {
		t.Fatal("command ran for invalid rectangle")
		return nil
	}, func(context.Context) (Rect, error) { return Rect{0, 0, 0, 5}, nil })
	if err == nil {
		t.Fatal("invalid selection accepted")
	}
}
