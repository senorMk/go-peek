//go:build !darwin || !cgo

package platform

func SetApplicationIcon(icon []byte) {}

func MoveWindowToCurrentDisplay() {}
