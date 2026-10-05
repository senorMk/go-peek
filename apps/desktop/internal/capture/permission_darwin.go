//go:build darwin && cgo

package capture

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics
#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CoreGraphics.h>

static int GoPeekCapturePermission(int request) {
    if (CGPreflightScreenCaptureAccess()) return 1;
    if (!request) return 0;
    __block bool granted = false;
    void (^work)(void) = ^{ granted = CGRequestScreenCaptureAccess(); };
    if ([NSThread isMainThread]) work();
    else dispatch_sync(dispatch_get_main_queue(), work);
    return granted || CGPreflightScreenCaptureAccess();
}
*/
import "C"

import "errors"

// EnsurePermission runs only in response to a user capture action, before hiding GoPeek.
func EnsurePermission() error {
	if C.GoPeekCapturePermission(1) != 0 {
		return nil
	}
	return errors.New("macOS has not granted screen recording access to this GoPeek build. In System Settings → Privacy & Security → Screen & System Audio Recording, remove the old GoPeek entry and add this .app again, then quit and reopen it")
}
