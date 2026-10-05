package main

import (
	"errors"

	"github.com/senorMk/go-peek/apps/desktop/internal/credentials"
	"github.com/senorMk/go-peek/apps/desktop/internal/prompts"
	"github.com/senorMk/go-peek/apps/desktop/internal/providers"
	"github.com/senorMk/go-peek/apps/desktop/internal/settings"
)

type Preferences struct {
	Config         settings.Config `json:"config"`
	KeyAvailable   bool            `json:"keyAvailable"`
	ImageSupported bool            `json:"imageSupported"`
	Message        string          `json:"message,omitempty"`
}

func (d *Desktop) GetPreferences() Preferences {
	<-d.ready
	d.requestMu.Lock()
	defer d.requestMu.Unlock()
	_, err := credentials.Get(d.config.Provider)
	message := d.settingsError
	if err != nil && !errors.Is(err, credentials.ErrMissing) {
		message = err.Error()
	}
	return Preferences{Config: d.config, KeyAvailable: err == nil, Message: message, ImageSupported: providers.SupportsImages(d.config.Model)}
}
func (d *Desktop) SavePreferences(cfg settings.Config) error {
	<-d.ready
	d.requestMu.Lock()
	defer d.requestMu.Unlock()
	if d.configPath == "" {
		return errors.New("local settings directory is unavailable")
	}
	// Provider edits must not overwrite a newer toolbar toggle with stale form state.
	cfg.AlwaysOnTop = d.config.AlwaysOnTop
	cfg.CaptureChecksEnabled = d.config.CaptureChecksEnabled
	if err := settings.Save(d.configPath, cfg); err != nil {
		return err
	}
	d.assistant.Reset()
	d.config, d.settingsError = cfg, ""
	return nil
}
func (d *Desktop) configuredProvider() (providers.Provider, error) {
	if err := d.config.Validate(); err != nil {
		return nil, err
	}
	key, err := credentials.Get(d.config.Provider)
	if err != nil {
		return nil, err
	}
	return providers.New(d.config.Provider, d.config.Model, key), nil
}
func (d *Desktop) StartResponse(problem prompts.Problem) (string, error) {
	<-d.ready
	if _, err := prompts.Build(problem); err != nil {
		return "", err
	}
	d.requestMu.Lock()
	defer d.requestMu.Unlock()
	if len(problem.Screenshots) > 0 && !providers.SupportsImages(d.config.Model) {
		return "", errors.New("image input is not enabled for this model; choose gpt-6-luna, gpt-6-sol, or gpt-6.1-sol, or explicitly send typed text only")
	}
	provider, err := d.configuredProvider()
	if err != nil {
		return "", err
	}
	return d.assistant.Start(d.ctx, problem, provider)
}
func (d *Desktop) AskFollowup(question string) (string, error) {
	<-d.ready
	d.requestMu.Lock()
	defer d.requestMu.Unlock()
	provider, err := d.configuredProvider()
	if err != nil {
		return "", err
	}
	return d.assistant.Followup(d.ctx, question, provider)
}
func (d *Desktop) DeleteKey(provider string) error {
	<-d.ready
	d.requestMu.Lock()
	defer d.requestMu.Unlock()
	if provider == d.config.Provider {
		d.assistant.Reset()
	}
	return credentials.Delete(provider)
}

// SaveKey accepts a newly entered secret; no binding returns a saved key.
func (d *Desktop) SaveKey(provider, key string) error {
	<-d.ready
	d.requestMu.Lock()
	defer d.requestMu.Unlock()
	if provider != d.config.Provider {
		return errors.New("save provider settings before adding its API key")
	}
	return credentials.Set(provider, key)
}

// SetCaptureChecks changes only diagnostic visibility, retaining provider and window settings.
func (d *Desktop) SetCaptureChecks(enabled bool) (settings.Config, error) {
	<-d.ready
	d.requestMu.Lock()
	defer d.requestMu.Unlock()
	if d.configPath == "" {
		return d.config, errors.New("local settings directory is unavailable")
	}
	cfg := d.config
	cfg.CaptureChecksEnabled = enabled
	if err := settings.Save(d.configPath, cfg); err != nil {
		return d.config, err
	}
	d.config, d.settingsError = cfg, ""
	return cfg, nil
}
