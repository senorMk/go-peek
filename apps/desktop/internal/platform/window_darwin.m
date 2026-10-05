//go:build darwin && cgo

#import <Cocoa/Cocoa.h>
#include "window_darwin.h"

// The Wails application delegate owns the main window even while it is hidden.
// Using that reference avoids selecting capture panels or file dialogs.
@protocol GoPeekWindowOwner
- (NSWindow *)mainWindow;
@end

static NSRect FrameForDisplay(NSRect frame, NSRect visible, BOOL sameDisplay) {
    frame.size.width = MIN(frame.size.width, visible.size.width);
    frame.size.height = MIN(frame.size.height, visible.size.height);
    if (!sameDisplay) {
        frame.origin.x = NSMidX(visible) - frame.size.width / 2;
        frame.origin.y = NSMidY(visible) - frame.size.height / 2;
    }
    // AppKit coordinates are points, including negative display origins. Use
    // visibleFrame to keep the window clear of the Dock and menu bar.
    frame.origin.x = MAX(NSMinX(visible), MIN(frame.origin.x, NSMaxX(visible) - frame.size.width));
    frame.origin.y = MAX(NSMinY(visible), MIN(frame.origin.y, NSMaxY(visible) - frame.size.height));
    return frame;
}

void GoPeekMoveWindowToCurrentDisplay(void) {
    // Never wait for Cocoa while the overlay controller holds its mutex.
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            id delegate = NSApp.delegate;
            if (![delegate respondsToSelector:@selector(mainWindow)]) return;
            NSWindow *window = [(id<GoPeekWindowOwner>)delegate mainWindow];
            if (!window || (window.styleMask & NSWindowStyleMaskFullScreen)) return;

            NSScreen *target = nil;
            NSPoint mouse = [NSEvent mouseLocation];
            for (NSScreen *screen in [NSScreen screens]) {
                if (NSPointInRect(mouse, screen.frame)) {
                    target = screen;
                    break;
                }
            }
            // mainScreen follows the key window; it is only a fallback because
            // GoPeek's previous key window may be on a different display.
            target = target ?: [NSScreen mainScreen] ?: [NSScreen screens].firstObject;
            if (!target) return;
            NSRect frame = FrameForDisplay(window.frame, target.visibleFrame, window.screen == target);
            [window setFrame:frame display:NO animate:NO];
        }
    });
}
