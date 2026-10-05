//go:build darwin && cgo

#import <Cocoa/Cocoa.h>
#import <Carbon/Carbon.h>
#include "shortcut_darwin.h"

extern void GoPeekShortcutReleased(int identifier);
static EventHotKeyRef gopeekShortcuts[2] = { NULL, NULL };
static EventHandlerRef gopeekHandler = NULL;
static const OSType gopeekSignature = 0x4750454B; // GPEK

static OSStatus HandleShortcut(EventHandlerCallRef next, EventRef event, void *data) {
    EventHotKeyID identifier;
    OSStatus status = GetEventParameter(event, kEventParamDirectObject,
        typeEventHotKeyID, NULL, sizeof(identifier), NULL, &identifier);
    if (status != noErr || identifier.signature != gopeekSignature || (identifier.id < 1 || identifier.id > 2)) {
        return eventNotHandledErr;
    }
    // Trigger on release so holding the key does not repeat the action.
    GoPeekShortcutReleased((int)identifier.id);
    return noErr;
}

static void OnMainThread(dispatch_block_t work) {
    if ([NSThread isMainThread]) { work(); }
    else { dispatch_sync(dispatch_get_main_queue(), work); }
}

int GoPeekRegisterShortcut(int id) {
    if (id < 1 || id > 2) return paramErr;
    __block OSStatus result = noErr;
    OnMainThread(^{
        if (gopeekShortcuts[id-1] != NULL) { result = eventHotKeyExistsErr; return; }
        if (gopeekHandler == NULL) {
            EventTypeSpec type = { kEventClassKeyboard, kEventHotKeyReleased };
            result = InstallApplicationEventHandler(HandleShortcut, 1, &type, NULL, &gopeekHandler);
            if (result != noErr) return;
        }
        EventHotKeyID identifier = { gopeekSignature, (UInt32)id };
        result = RegisterEventHotKey(id == 1 ? kVK_ANSI_B : kVK_ANSI_S,
            id == 1 ? cmdKey : (cmdKey | shiftKey), identifier,
            GetApplicationEventTarget(), 0, &gopeekShortcuts[id-1]);
        if (result != noErr && gopeekShortcuts[0] == NULL && gopeekShortcuts[1] == NULL) {
            RemoveEventHandler(gopeekHandler);
            gopeekHandler = NULL;
        }
    });
    return (int)result;
}

void GoPeekUnregisterShortcut(int id) {
    if (id < 1 || id > 2) return;
    OnMainThread(^{
        if (gopeekShortcuts[id-1] != NULL) {
            UnregisterEventHotKey(gopeekShortcuts[id-1]);
            gopeekShortcuts[id-1] = NULL;
        }
        if (gopeekHandler != NULL && gopeekShortcuts[0] == NULL && gopeekShortcuts[1] == NULL) {
            RemoveEventHandler(gopeekHandler);
            gopeekHandler = NULL;
        }
    });
}
