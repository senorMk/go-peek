//go:build !darwin || !cgo

package credentials

import "errors"

func Get(string) (string, error) {
	return "", errors.New("credential storage requires native macOS Keychain support")
}
func Set(string, string) error {
	return errors.New("credential storage requires native macOS Keychain support")
}
func Delete(string) error {
	return errors.New("credential storage requires native macOS Keychain support")
}
