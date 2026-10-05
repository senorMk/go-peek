//go:build darwin && cgo

#import <Cocoa/Cocoa.h>
#include "region_darwin.h"

// Selection border dimensions in macOS points. Edit these and run make prototype.
static const CGFloat GoPeekRegionBorderWidth = 2.0;
static const CGFloat GoPeekRegionDashLength = 36.0;
static const CGFloat GoPeekRegionDashGap = 32.0;

// All selector state is owned by Cocoa's main thread. No event taps are used.
static NSMutableArray *panels;
static id keyMonitor;
static id screenObserver;
static id resignObserver;
static GoPeekRegion result;
static BOOL cursorPushed;

static void OnMain(dispatch_block_t work) {
    if ([NSThread isMainThread]) work();
    else dispatch_sync(dispatch_get_main_queue(), work);
}

static void Finish(int state) {
    result.state = state;
    for (NSWindow *panel in panels) [panel orderOut:nil];
    if (cursorPushed) { [NSCursor pop]; cursorPushed = NO; }
}

static void Cleanup(void) {
    if (keyMonitor) { [NSEvent removeMonitor:keyMonitor]; keyMonitor = nil; }
    if (screenObserver) { [[NSNotificationCenter defaultCenter] removeObserver:screenObserver]; screenObserver = nil; }
    if (resignObserver) { [[NSNotificationCenter defaultCenter] removeObserver:resignObserver]; resignObserver = nil; }
    for (NSWindow *panel in panels) [panel close];
    [panels release]; panels = nil;
    if (cursorPushed) { [NSCursor pop]; cursorPushed = NO; }
}

@interface GoPeekRegionPanel : NSPanel
@end
@implementation GoPeekRegionPanel
- (BOOL)canBecomeKeyWindow { return YES; }
@end

@interface GoPeekRegionView : NSView {
    NSPoint anchor;
    NSRect selection;
    BOOL dragging;
}
@end

@implementation GoPeekRegionView
- (BOOL)isFlipped { return YES; }
- (BOOL)isOpaque { return NO; }
- (BOOL)acceptsFirstResponder { return YES; }
- (BOOL)acceptsFirstMouse:(NSEvent *)event { return YES; }
- (void)resetCursorRects { [self addCursorRect:self.bounds cursor:[NSCursor crosshairCursor]]; }
- (NSPoint)pointForEvent:(NSEvent *)event {
    NSPoint p = [self convertPoint:event.locationInWindow fromView:nil];
    p.x = MAX(0, MIN(self.bounds.size.width, p.x));
    p.y = MAX(0, MIN(self.bounds.size.height, p.y));
    return p;
}
- (void)mouseDown:(NSEvent *)event {
    if (result.state != 0) return;
    anchor = [self pointForEvent:event];
    selection = NSMakeRect(anchor.x, anchor.y, 0, 0);
    dragging = YES;
    [self.window makeKeyWindow];
    [self setNeedsDisplay:YES];
}
- (void)mouseDragged:(NSEvent *)event {
    if (!dragging || result.state != 0) return;
    NSPoint p = [self pointForEvent:event];
    selection = NSMakeRect(MIN(anchor.x,p.x), MIN(anchor.y,p.y), fabs(p.x-anchor.x), fabs(p.y-anchor.y));
    [self setNeedsDisplay:YES];
}
- (void)mouseUp:(NSEvent *)event {
    if (!dragging || result.state != 0) return;
    [self mouseDragged:event];
    dragging = NO;
    if (selection.size.width < 2 || selection.size.height < 2) {
        selection = NSZeroRect;
        [self setNeedsDisplay:YES];
        return;
    }
    // Flipped local coordinates -> AppKit desktop -> Core Graphics top-left.
    NSRect frame = self.window.frame;
    CGFloat primaryTop = NSMaxY([NSScreen screens].firstObject.frame);
    result.x = frame.origin.x + selection.origin.x;
    result.y = primaryTop - NSMaxY(frame) + selection.origin.y;
    result.width = selection.size.width;
    result.height = selection.size.height;
    Finish(1);
}
- (void)drawRect:(NSRect)dirty {
    [[NSColor colorWithWhite:0 alpha:0.18] setFill];
    NSRectFillUsingOperation(self.bounds, NSCompositingOperationCopy);
    if (selection.size.width >= 2 && selection.size.height >= 2) {
        NSRectFillUsingOperation(selection, NSCompositingOperationClear);
        CGFloat inset = MIN(GoPeekRegionBorderWidth / 2.0, MIN(selection.size.width, selection.size.height) / 4.0);
        NSBezierPath *outline = [NSBezierPath bezierPathWithRect:NSInsetRect(selection, inset, inset)];
        CGFloat dashes[] = { GoPeekRegionDashLength, GoPeekRegionDashGap };
        [outline setLineDash:dashes count:2 phase:0];
        outline.lineWidth = GoPeekRegionBorderWidth;
        outline.lineCapStyle = NSLineCapStyleRound;
        [[NSColor colorWithSRGBRed:0.2 green:1 blue:0.45 alpha:1] setStroke];
        [outline stroke];
    }
}
@end

int GoPeekBeginRegion(void) {
    __block int started = 0;
    OnMain(^{
        @autoreleasepool {
            if (panels) return;
            NSArray *screens = [NSScreen screens];
            if (screens.count == 0) return;
            result = (GoPeekRegion){0};
            panels = [[NSMutableArray alloc] init];
            NSPoint mouse = [NSEvent mouseLocation];
            NSWindow *keyPanel = nil;
            for (NSScreen *screen in screens) {
                GoPeekRegionPanel *panel = [[GoPeekRegionPanel alloc] initWithContentRect:screen.frame
                    styleMask:NSWindowStyleMaskBorderless backing:NSBackingStoreBuffered defer:NO];
                [panel setFrame:screen.frame display:NO];
                panel.releasedWhenClosed = NO;
                panel.opaque = NO;
                panel.backgroundColor = [NSColor clearColor];
                panel.hasShadow = NO;
                panel.level = NSScreenSaverWindowLevel;
                panel.sharingType = NSWindowSharingNone;
                panel.hidesOnDeactivate = NO;
                panel.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorFullScreenAuxiliary | NSWindowCollectionBehaviorStationary;
                GoPeekRegionView *view = [[GoPeekRegionView alloc] initWithFrame:NSMakeRect(0,0,screen.frame.size.width,screen.frame.size.height)];
                panel.contentView = view;
                [panel makeFirstResponder:view];
                [view release];
                [panels addObject:panel];
                if (NSPointInRect(mouse,screen.frame)) keyPanel = panel;
                [panel release];
            }
            [NSApp activateIgnoringOtherApps:YES];
            for (NSWindow *panel in panels) [panel orderFrontRegardless];
            [(keyPanel ?: panels.firstObject) makeKeyWindow];
            [[NSCursor crosshairCursor] push]; cursorPushed = YES;
            keyMonitor = [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskKeyDown handler:^NSEvent *(NSEvent *event) {
                if (event.keyCode == 53 && result.state == 0) { Finish(2); return nil; }
                return event;
            }];
            screenObserver = [[NSNotificationCenter defaultCenter] addObserverForName:NSApplicationDidChangeScreenParametersNotification object:nil queue:nil usingBlock:^(NSNotification *note) { if (result.state == 0) Finish(2); }];
            resignObserver = [[NSNotificationCenter defaultCenter] addObserverForName:NSApplicationDidResignActiveNotification object:nil queue:nil usingBlock:^(NSNotification *note) { if (result.state == 0) Finish(2); }];
            started = 1;
        }
    });
    return started;
}

GoPeekRegion GoPeekPollRegion(void) {
    __block GoPeekRegion snapshot;
    OnMain(^{ snapshot = result; if (result.state != 0) Cleanup(); });
    return snapshot;
}

void GoPeekCancelRegion(void) {
    OnMain(^{ Finish(2); Cleanup(); });
}
