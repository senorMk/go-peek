package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsRoundTripAndInvalidZenProtocol(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "settings.json")
	cfg, err := Load(path)
	if err != nil || cfg != Default() {
		t.Fatal("missing settings did not use defaults")
	}
	cfg.Provider = "zen"
	cfg.AlwaysOnTop = true
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got != cfg {
		t.Fatalf("roundtrip: %+v %v", got, err)
	}
	stat, err := os.Stat(path)
	if err != nil || stat.Mode().Perm() != 0600 {
		t.Fatal("settings file permissions are not private")
	}
	cfg.Model = "claude-fable-5"
	if err := Save(path, cfg); err == nil {
		t.Fatal("non-Responses Zen model accepted")
	}
	old, _ := Load(path)
	if old != got {
		t.Fatal("invalid settings overwrote valid settings")
	}
}
func TestRejectSecretFieldsAndUnknownVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	for _, body := range []string{`{"version":1,"provider":"openai","model":"gpt-6-luna","apiKey":"secret"}`, `{"version":99,"provider":"openai","model":"model"}`, `{"version":1,"provider":"openai","model":"model"} {}`} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatal("invalid settings accepted")
		}
	}
}

func TestLegacySettingsAndPinnedPreferenceAcrossReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"provider":"zen","model":"gpt-6-sol"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil || cfg.AlwaysOnTop || cfg.CaptureChecksEnabled {
		t.Fatalf("legacy settings failed: %+v %v", cfg, err)
	}
	for _, enabled := range []bool{true, false, true} {
		cfg.AlwaysOnTop = enabled
		if err := Save(path, cfg); err != nil {
			t.Fatal(err)
		}
		reopened, err := Load(path)
		if err != nil || reopened.AlwaysOnTop != enabled || reopened.Provider != "zen" || reopened.Model != "gpt-6-sol" {
			t.Fatalf("reopened settings: %+v %v", reopened, err)
		}
		cfg = reopened
	}
}
