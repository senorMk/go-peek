package prompts

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image/png"
	"strings"
)

const MaxScreenshots = 20
const MaxImageBytes = 20 * 1024 * 1024
const MaxImagesBytes = 32 * 1024 * 1024
const MaxImagesEncodedBytes = (MaxImagesBytes+2)/3*4 + MaxScreenshots*len("data:image/png;base64,")
const MaxImagesPixels = 64_000_000

// ValidateScreenshots accepts only bounded inline PNGs, never remote URLs or files.
// Dimensions come from the PNG header, not frontend-supplied metadata.
func ValidateScreenshots(images []string) error {
	if len(images) > MaxScreenshots {
		return errors.New("at most 20 screenshots may be submitted")
	}
	totalBytes, totalPixels := 0, int64(0)
	for _, image := range images {
		const prefix = "data:image/png;base64,"
		if !strings.HasPrefix(image, prefix) {
			return errors.New("screenshots must be inline PNG images")
		}
		encoded := image[len(prefix):]
		if len(encoded) > base64.StdEncoding.EncodedLen(MaxImageBytes) {
			return errors.New("a screenshot exceeds the 20 MiB limit; capture a smaller region")
		}
		if len(encoded) > (MaxImagesBytes-totalBytes+2)/3*4 {
			return errors.New("screenshots exceed the 32 MiB request limit; remove images or capture smaller regions")
		}
		data, err := base64.StdEncoding.Strict().DecodeString(encoded)
		if err != nil {
			return errors.New("a screenshot has invalid image encoding")
		}
		totalBytes += len(data)
		if len(data) > MaxImageBytes || totalBytes > MaxImagesBytes {
			return errors.New("screenshots exceed the image request limit; remove images or capture smaller regions")
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
			return errors.New("a screenshot is not a valid PNG image")
		}
		totalPixels += int64(cfg.Width) * int64(cfg.Height)
		if totalPixels > MaxImagesPixels {
			return errors.New("screenshots exceed 64 total megapixels; capture smaller regions")
		}
	}
	return nil
}
