#include "../window_darwin.m"
#include <stdio.h>

static int failures;

static void Check(const char *name, NSRect original, NSRect visible, BOOL sameDisplay, NSRect expected) {
    NSRect actual = FrameForDisplay(original, visible, sameDisplay);
    if (!NSEqualRects(actual, expected)) {
        fprintf(stderr, "%s: expected %s, got %s\n", name,
            NSStringFromRect(expected).UTF8String, NSStringFromRect(actual).UTF8String);
        failures++;
    }
}

int main(void) {
    @autoreleasepool {
        NSRect window = NSMakeRect(100, 100, 520, 620);
        Check("same display preserves position", window, NSMakeRect(0, 40, 1440, 840), YES, window);
        Check("display to the right", window, NSMakeRect(1440, 40, 1920, 1040), NO, NSMakeRect(2140, 250, 520, 620));
        Check("display to the left", window, NSMakeRect(-1920, 40, 1920, 1040), NO, NSMakeRect(-1220, 250, 520, 620));
        Check("display above", window, NSMakeRect(0, 900, 1440, 900), NO, NSMakeRect(460, 1040, 520, 620));
        Check("display below", window, NSMakeRect(0, -900, 1440, 860), NO, NSMakeRect(460, -780, 520, 620));
        Check("disconnected display clamps old position", NSMakeRect(-1500, 1000, 520, 620), NSMakeRect(0, 40, 1440, 840), YES, NSMakeRect(0, 260, 520, 620));
        Check("oversized window fits usable area", NSMakeRect(0, 0, 1800, 1200), NSMakeRect(-1000, 40, 1000, 700), NO, NSMakeRect(-1000, 40, 1000, 700));
    }
    return failures ? 1 : 0;
}
