//go:build !darwin || !cgo

package platform

import "errors"

func RegisterShortcut() (<-chan struct{}, func(), error) {
	return nil, nil, errors.New("global shortcut requires the macOS build with native extensions enabled")
}

func RegisterCaptureShortcut() (<-chan struct{}, func(), error) { return RegisterShortcut() }
