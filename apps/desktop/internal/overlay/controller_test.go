package overlay

import (
	"sync"
	"testing"
	"time"
)

func fixture() (*Controller, chan string) {
	events := make(chan string, 1000)
	c := New(func() { events <- "show" }, func() { events <- "hide" })
	return c, events
}
func expect(t *testing.T, events <-chan string, expected string) {
	t.Helper()
	select {
	case got := <-events:
		if got != expected {
			t.Fatalf("wanted %s, got %s", expected, got)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", expected)
	}
}
func quiet(t *testing.T, events <-chan string, duration time.Duration) {
	t.Helper()
	select {
	case got := <-events:
		t.Fatalf("unexpected window action: %s", got)
	case <-time.After(duration):
	}
}

func TestTimedHideRestoresAndToggleUsesRestoredState(t *testing.T) {
	c, events := fixture()
	defer c.Close()
	c.HideFor(20 * time.Millisecond)
	expect(t, events, "hide")
	expect(t, events, "show")
	c.Toggle()
	expect(t, events, "hide")
	c.Toggle()
	expect(t, events, "show")
}

func TestShortcutToggleInvalidatesPendingRestore(t *testing.T) {
	c, events := fixture()
	defer c.Close()
	c.HideFor(50 * time.Millisecond)
	expect(t, events, "hide")
	c.Toggle()
	expect(t, events, "show")
	c.Toggle()
	expect(t, events, "hide")
	quiet(t, events, 100*time.Millisecond)
}

func TestShortcutRestoreUsesCurrentDisplayWithoutChangingOtherRestores(t *testing.T) {
	c, events := fixture()
	defer c.Close()
	showHere := func() { events <- "show-current-display" }
	c.ToggleWithRestore(showHere)
	expect(t, events, "hide")
	c.ToggleWithRestore(showHere)
	expect(t, events, "show-current-display")
	c.HideFor(20 * time.Millisecond)
	expect(t, events, "hide")
	expect(t, events, "show")
	finish, ok := c.BeginCapture()
	if !ok {
		t.Fatal("capture refused")
	}
	expect(t, events, "hide")
	c.ToggleWithRestore(showHere)
	quiet(t, events, 20*time.Millisecond)
	finish()
	expect(t, events, "show")
	c.Close()
	c.ToggleWithRestore(showHere)
	quiet(t, events, 20*time.Millisecond)
}

func TestCurrentDisplayRestoreCancelsTimedRestore(t *testing.T) {
	c, events := fixture()
	defer c.Close()
	c.HideFor(50 * time.Millisecond)
	expect(t, events, "hide")
	c.ToggleWithRestore(func() { events <- "show-current-display" })
	expect(t, events, "show-current-display")
	c.Toggle()
	expect(t, events, "hide")
	quiet(t, events, 100*time.Millisecond)
}

func TestRepeatedTimedHideReplacesRestoreDeadline(t *testing.T) {
	c, events := fixture()
	defer c.Close()
	c.HideFor(50 * time.Millisecond)
	expect(t, events, "hide")
	c.HideFor(250 * time.Millisecond)
	expect(t, events, "hide")
	quiet(t, events, 100*time.Millisecond)
	expect(t, events, "show")
}

func TestShutdownPreventsWindowActions(t *testing.T) {
	c, events := fixture()
	c.HideFor(20 * time.Millisecond)
	expect(t, events, "hide")
	c.Close()
	c.Toggle()
	c.HideFor(time.Millisecond)
	c.Close()
	quiet(t, events, 50*time.Millisecond)
}

func TestConcurrentWindowControlsAndShutdown(t *testing.T) {
	c, _ := fixture()
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Go(func() { c.Toggle(); c.HideFor(time.Second) })
	}
	wg.Go(c.Close)
	wg.Wait()
	c.Close()
}

func TestCaptureHidesUntilFinishedAndSuspendsToggles(t *testing.T) {
	c, events := fixture()
	defer c.Close()
	c.HideFor(20 * time.Millisecond)
	expect(t, events, "hide")
	finish, ok := c.BeginCapture()
	if !ok {
		t.Fatal("capture refused")
	}
	expect(t, events, "hide")
	c.Toggle()
	c.HideFor(time.Millisecond)
	if _, ok := c.BeginCapture(); ok {
		t.Fatal("overlapping capture allowed")
	}
	quiet(t, events, 40*time.Millisecond)
	finish()
	finish()
	expect(t, events, "show")
	c.Toggle()
	expect(t, events, "hide")
}

func TestCaptureFinishAfterShutdownDoesNotShowWindow(t *testing.T) {
	c, events := fixture()
	finish, ok := c.BeginCapture()
	if !ok {
		t.Fatal("capture refused")
	}
	expect(t, events, "hide")
	c.Close()
	finish()
	quiet(t, events, 20*time.Millisecond)
}
