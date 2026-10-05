//go:build darwin && cgo

package capture

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework Cocoa
#include "region_darwin.h"
*/
import "C"

import (
	"context"
	"errors"
	"time"
)

func selectRegion(ctx context.Context) (Rect, error) {
	if ctx.Err() != nil {
		return Rect{}, ctx.Err()
	}
	if C.GoPeekBeginRegion() == 0 {
		return Rect{}, errors.New("Could not open the region selector")
	}
	defer C.GoPeekCancelRegion()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return Rect{}, ctx.Err()
		case <-ticker.C:
			result := C.GoPeekPollRegion()
			switch result.state {
			case 1:
				return Rect{float64(result.x), float64(result.y), float64(result.width), float64(result.height)}, nil
			case 2:
				return Rect{}, ErrCancelled
			}
		}
	}
}
