//go:build !darwin || !cgo

package capture

import "errors"

func EnsurePermission() error {
	return errors.New("Screenshot capture requires the macOS build with native extensions enabled")
}
