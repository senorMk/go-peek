//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework Cocoa
#include "window_darwin.h"
*/
import "C"

// MoveWindowToCurrentDisplay queues placement on Cocoa's main thread. Call
// immediately before WindowShow so placement runs before the window appears.
func MoveWindowToCurrentDisplay() {
	C.GoPeekMoveWindowToCurrentDisplay()
}
