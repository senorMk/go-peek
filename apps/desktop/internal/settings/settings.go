package settings

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type Config struct {
	Version              int    `json:"version"`
	Provider             string `json:"provider"`
	Model                string `json:"model"`
	AlwaysOnTop          bool   `json:"alwaysOnTop"`
	CaptureChecksEnabled bool   `json:"captureChecksEnabled"`
}

func Default() Config { return Config{Version: 1, Provider: "openai", Model: "gpt-6-luna"} }

var modelID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$`)

func (c Config) Validate() error {
	if c.Version != 1 {
		return errors.New("unsupported settings version")
	}
	if c.Provider != "openai" && c.Provider != "zen" {
		return errors.New("select OpenAI or OpenCode Zen")
	}
	if !modelID.MatchString(c.Model) {
		return errors.New("enter a valid model ID")
	}
	if c.Provider == "zen" && c.Model != "gpt-6-luna" && c.Model != "gpt-6-sol" && c.Model != "gpt-6.1-sol" {
		return errors.New("Zen currently supports gpt-6-luna, gpt-6-sol, and gpt-6.1-sol through Responses")
	}
	return nil
}
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", errors.New("cannot locate the local settings directory")
	}
	return filepath.Join(dir, "go-peek", "settings.json"), nil
}
func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Config{}, errors.New("cannot read settings")
	}
	defer f.Close()
	decoder := json.NewDecoder(io.LimitReader(f, 8193))
	decoder.DisallowUnknownFields()
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, errors.New("settings are invalid; save provider settings to replace them")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Config{}, errors.New("settings contain unexpected extra data")
	}
	return cfg, cfg.Validate()
}
func Save(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return errors.New("cannot create settings directory")
	}
	f, err := os.CreateTemp(filepath.Dir(path), "settings-*")
	if err != nil {
		return errors.New("cannot create settings file")
	}
	defer os.Remove(f.Name())
	if err := json.NewEncoder(f).Encode(cfg); err != nil {
		f.Close()
		return errors.New("cannot encode settings")
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return errors.New("cannot write settings")
	}
	if err := f.Close(); err != nil {
		return errors.New("cannot close settings file")
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return errors.New("cannot save settings")
	}
	return nil
}
