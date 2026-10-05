//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework Cocoa -framework Carbon
#include "shortcut_darwin.h"
*/
import "C"

import (
	"fmt"
	"sync"
)

// Each app-owned shortcut has a bounded channel; notifications never block Cocoa.
var shortcutEvents = [2]chan struct{}{make(chan struct{}, 1), make(chan struct{}, 1)}

//export GoPeekShortcutReleased
func GoPeekShortcutReleased(identifier C.int) {
	if identifier < 1 || identifier > 2 {
		return
	}
	select {
	case shortcutEvents[int(identifier)-1] <- struct{}{}:
	default:
	}
}

// RegisterShortcut uses the existing Cocoa main loop, without a keyboard event tap.
// Call after the desktop is ready. Cleanup removes both the hotkey and its event handler.
func RegisterShortcut() (<-chan struct{}, func(), error) {
	return registerShortcut(1, ShortcutLabel)
}

func RegisterCaptureShortcut() (<-chan struct{}, func(), error) {
	return registerShortcut(2, CaptureShortcutLabel)
}

func registerShortcut(id int, label string) (<-chan struct{}, func(), error) {
	if status := int(C.GoPeekRegisterShortcut(C.int(id))); status != 0 {
		return nil, nil, fmt.Errorf("could not register %s (macOS status %d); another app may use it. Use the app buttons", label, status)
	}
	var once sync.Once
	return shortcutEvents[id-1], func() { once.Do(func() { C.GoPeekUnregisterShortcut(C.int(id)) }) }, nil
}
