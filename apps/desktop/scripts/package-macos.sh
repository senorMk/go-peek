#!/usr/bin/env bash
set -euo pipefail
if [[ "$(go env GOOS)" != "darwin" ]]; then
  echo "The feasibility bundle requires macOS and Xcode Command Line Tools." >&2
  exit 1
fi
bundle="build/bin/Assessment Caddy.app"
mkdir -p "$bundle/Contents/MacOS"
CGO_LDFLAGS="${CGO_LDFLAGS:-} -framework UniformTypeIdentifiers" go build -tags desktop,production -ldflags '-s -w' -o "$bundle/Contents/MacOS/assessment-caddy" .
cp build/Info.plist "$bundle/Contents/Info.plist"
# Sign the completed bundle ad hoc after adding its resources, as Wails does.
codesign --force --sign - "$bundle"
codesign --verify --deep --strict "$bundle"
echo "Built $bundle (ad hoc signed local feasibility bundle)."
