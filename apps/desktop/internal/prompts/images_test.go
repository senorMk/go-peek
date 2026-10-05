package prompts

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"strings"
	"testing"
)

func pngFixture(t *testing.T) string {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

func TestScreenshotOnlyProblemKeepsPixelsOutOfTextPrompt(t *testing.T) {
	image := pngFixture(t)
	for _, action := range []string{"hint", "review", "debug"} {
		text, err := Build(Problem{Screenshots: []string{image, image}, Language: "Go", Action: action})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(text, "base64") || !strings.Contains(text, `"screenshotCount":2`) {
			t.Fatal("screenshots must be attached separately from text")
		}
	}
}

func TestRejectInvalidAndExcessScreenshotInput(t *testing.T) {
	for _, input := range []string{"https://example.test/image.png", "file:///private.png", "data:image/jpeg;base64,AAAA", "data:image/png;base64,invalid!", "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not PNG"))} {
		if err := ValidateScreenshots([]string{input}); err == nil {
			t.Fatal("invalid screenshot accepted")
		}
	}
	image := pngFixture(t)
	images := make([]string, MaxScreenshots+1)
	for i := range images {
		images[i] = image
	}
	if err := ValidateScreenshots(images); err == nil {
		t.Fatal("image count unbounded")
	}
	oversized := "data:image/png;base64," + strings.Repeat("A", base64.StdEncoding.EncodedLen(MaxImageBytes)+4)
	if err := ValidateScreenshots([]string{oversized}); err == nil {
		t.Fatal("image bytes unbounded")
	}
	if _, err := Build(Problem{Language: "Go", Action: "hint"}); err == nil {
		t.Fatal("empty problem accepted")
	}
}
