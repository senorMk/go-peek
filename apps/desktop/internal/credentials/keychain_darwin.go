//go:build darwin && cgo

package credentials

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework Security
#import <Foundation/Foundation.h>
#import <Security/Security.h>
#include <stdlib.h>
#include <string.h>

static NSDictionary *GoPeekKeyQuery(const char *account) {
    return @{ (id)kSecClass: (id)kSecClassGenericPassword,
        (id)kSecAttrService: @"com.senormk.go-peek.providers",
        (id)kSecAttrAccount: [NSString stringWithUTF8String:account] };
}
static int GoPeekGetKey(const char *account, CFDataRef *result) {
    @autoreleasepool {
        NSMutableDictionary *query = [GoPeekKeyQuery(account) mutableCopy];
        query[(id)kSecReturnData] = @YES;
        query[(id)kSecMatchLimit] = (id)kSecMatchLimitOne;
        OSStatus status = SecItemCopyMatching((CFDictionaryRef)query, (CFTypeRef *)result);
        [query release];
        return (int)status;
    }
}
static int GoPeekSetKey(const char *account, void *bytes, int length) {
    @autoreleasepool {
        NSDictionary *query = GoPeekKeyQuery(account);
        NSData *data = [NSData dataWithBytes:bytes length:length];
        NSDictionary *value = @{ (id)kSecValueData: data };
        OSStatus status = SecItemUpdate((CFDictionaryRef)query, (CFDictionaryRef)value);
        if (status == errSecItemNotFound) {
            NSMutableDictionary *item = [query mutableCopy];
            item[(id)kSecValueData] = data;
            item[(id)kSecAttrAccessible] = (id)kSecAttrAccessibleWhenUnlockedThisDeviceOnly;
            status = SecItemAdd((CFDictionaryRef)item, NULL);
            [item release];
        }
        memset(bytes, 0, length);
        return (int)status;
    }
}
static int GoPeekDeleteKey(const char *account) {
    @autoreleasepool {
        OSStatus status = SecItemDelete((CFDictionaryRef)GoPeekKeyQuery(account));
        return (int)status;
    }
}
*/
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

func Get(provider string) (string, error) {
	if !valid(provider) {
		return "", errors.New("invalid credential provider")
	}
	account := C.CString(provider)
	defer C.free(unsafe.Pointer(account))
	var result C.CFDataRef
	status := C.GoPeekGetKey(account, &result)
	if status == C.errSecItemNotFound {
		return "", ErrMissing
	}
	if status != 0 {
		return "", fmt.Errorf("Keychain access failed (status %d); unlock your login Keychain and retry", status)
	}
	defer C.CFRelease(C.CFTypeRef(result))
	return string(C.GoBytes(unsafe.Pointer(C.CFDataGetBytePtr(result)), C.int(C.CFDataGetLength(result)))), nil
}
func Set(provider, key string) error {
	if !valid(provider) {
		return errors.New("invalid credential provider")
	}
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 4096 || strings.ContainsAny(key, "\r\n") {
		return errors.New("API key must be a nonempty single line, at most 4096 bytes")
	}
	account := C.CString(provider)
	defer C.free(unsafe.Pointer(account))
	bytes := C.CBytes([]byte(key))
	defer C.free(bytes)
	if status := C.GoPeekSetKey(account, bytes, C.int(len(key))); status != 0 {
		return fmt.Errorf("could not save API key to Keychain (status %d)", status)
	}
	return nil
}
func Delete(provider string) error {
	if !valid(provider) {
		return errors.New("invalid credential provider")
	}
	account := C.CString(provider)
	defer C.free(unsafe.Pointer(account))
	status := C.GoPeekDeleteKey(account)
	if status != 0 && status != C.errSecItemNotFound {
		return fmt.Errorf("could not delete API key (Keychain status %d)", status)
	}
	return nil
}
