//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework Cocoa
#include "window_darwin.h"
*/
import "C"

import "unsafe"

// SetApplicationIcon also supplies the Dock icon for direct executable and dev
// launches, which do not have the packaged app's Info.plist and ICNS resource.
func SetApplicationIcon(icon []byte) {
	if len(icon) == 0 {
		return
	}
	C.GoPeekSetApplicationIcon((*C.uchar)(unsafe.Pointer(&icon[0])), C.int(len(icon)))
}

// MoveWindowToCurrentDisplay queues placement on Cocoa's main thread. Call
// immediately before WindowShow so placement runs before the window appears.
func MoveWindowToCurrentDisplay() {
	C.GoPeekMoveWindowToCurrentDisplay()
}
