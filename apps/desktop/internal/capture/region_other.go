//go:build !darwin || !cgo

package capture

import (
	"context"
	"errors"
)

func selectRegion(context.Context) (Rect, error) {
	return Rect{}, errors.New("Custom region selection requires the macOS native build")
}
