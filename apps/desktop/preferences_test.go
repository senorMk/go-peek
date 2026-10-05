package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/senorMk/go-peek/apps/desktop/internal/session"
	"github.com/senorMk/go-peek/apps/desktop/internal/settings"
)

func testDesktop(t *testing.T) *Desktop {
	t.Helper()
	ready := make(chan struct{})
	close(ready)
	return &Desktop{ready: ready, config: settings.Default(), configPath: filepath.Join(t.TempDir(), "settings.json"), assistant: session.NewAssistant(func(session.Event) {})}
}

func TestProviderEditPreservesNewerAlwaysOnTopPreference(t *testing.T) {
	d := testDesktop(t)
	d.config.AlwaysOnTop = true
	d.config.CaptureChecksEnabled = true
	staleForm := settings.Default()
	staleForm.Provider = "zen"
	staleForm.Model = "gpt-6-sol"
	if err := d.SavePreferences(staleForm); err != nil {
		t.Fatal(err)
	}
	reopened, err := settings.Load(d.configPath)
	if err != nil || !reopened.AlwaysOnTop || !reopened.CaptureChecksEnabled || reopened.Provider != "zen" || reopened.Model != "gpt-6-sol" {
		t.Fatalf("provider edit overwrote window settings: %+v %v", reopened, err)
	}
}

func TestFailedPinSaveDoesNotChangeWindowOrInMemoryPreference(t *testing.T) {
	d := testDesktop(t)
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("file instead of directory"), 0600); err != nil {
		t.Fatal(err)
	}
	d.configPath = filepath.Join(blocker, "settings.json")
	// No desktop runtime context: any attempt to change the native window before
	// a successful save would fail this test.
	if err := d.SetAlwaysOnTop(true); err == nil {
		t.Fatal("failed persistence reported success")
	}
	if d.config.AlwaysOnTop || d.GetStatus().AlwaysOnTop {
		t.Fatal("failed save changed the preference")
	}
}

func TestCaptureChecksVisibilityPersistsWithoutChangingOtherPreferences(t *testing.T) {
	d := testDesktop(t)
	d.config.AlwaysOnTop = true
	d.config.Provider = "zen"
	for _, enabled := range []bool{true, false} {
		cfg, err := d.SetCaptureChecks(enabled)
		if err != nil || cfg.CaptureChecksEnabled != enabled {
			t.Fatalf("toggle failed: %+v %v", cfg, err)
		}
		reopened, err := settings.Load(d.configPath)
		if err != nil || reopened.CaptureChecksEnabled != enabled || !reopened.AlwaysOnTop || reopened.Provider != "zen" {
			t.Fatalf("preference lost: %+v %v", reopened, err)
		}
	}
	d.configPath = ""
	if _, err := d.SetCaptureChecks(true); err == nil || d.config.CaptureChecksEnabled {
		t.Fatal("failed save changed diagnostic visibility")
	}
}

func TestSaveKeyRejectsWrongProviderAndInvalidInputBeforeKeychain(t *testing.T) {
	d := testDesktop(t)
	if err := d.SaveKey("zen", "test-key"); err == nil {
		t.Fatal("accepted a key for an unsaved provider")
	}
	for _, key := range []string{"", "   ", "test\nkey"} {
		if err := d.SaveKey("openai", key); err == nil {
			t.Fatal("accepted invalid credential input")
		}
	}
	if d.config != settings.Default() {
		t.Fatal("key validation changed provider settings")
	}
}
