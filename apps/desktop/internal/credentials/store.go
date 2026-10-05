package credentials

import "errors"

var ErrMissing = errors.New("no API key saved; add your key in Settings → Provider")

func valid(provider string) bool { return provider == "openai" || provider == "zen" }
