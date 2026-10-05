//go:build darwin && cgo

package platform

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// Exercise the actual AppKit placement math without starting an app or moving
// real windows. A cgo build already requires the native compiler and Cocoa SDK.
func TestNativeWindowPlacement(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "window-placement")
	cmd := exec.Command("clang", "-fblocks", "-framework", "Cocoa", "testdata/window_placement.m", "-o", binary)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile native placement checks: %v\n%s", err, output)
	}
	if output, err := exec.Command(binary).CombinedOutput(); err != nil {
		t.Fatalf("native placement checks: %v\n%s", err, output)
	}
}
