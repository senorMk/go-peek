package prompts

import (
	_ "embed"
	"encoding/json"
	"slices"
)

// languages.json is also imported by the frontend selector so both use one catalog.
//
//go:embed languages.json
var languageCatalog []byte

var languages = func() []string {
	var names []string
	if err := json.Unmarshal(languageCatalog, &names); err != nil {
		panic("invalid embedded language catalog")
	}
	return names
}()

func supportsLanguage(language string) bool { return slices.Contains(languages, language) }
