#!/usr/bin/env bash
set -euo pipefail
if [[ "$(go env GOOS)" != "darwin" ]]; then
  echo "The feasibility bundle requires macOS and Xcode Command Line Tools." >&2
  exit 1
fi
bundle="build/bin/GoPeek.app"
iconset="build/bin/GoPeek.iconset"
mkdir -p "$bundle/Contents/MacOS" "$bundle/Contents/Resources" "$iconset"
# appicon.png is the 1024px rendering of build/appicon.svg. Generate every
# standard and Retina representation with native macOS tools before signing.
for size in 16 32 128 256 512; do
  sips -z "$size" "$size" build/appicon.png --out "$iconset/icon_${size}x${size}.png" >/dev/null
  retina_size=$((size * 2))
  sips -z "$retina_size" "$retina_size" build/appicon.png --out "$iconset/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$iconset" -o "$bundle/Contents/Resources/appicon.icns"
CGO_LDFLAGS="${CGO_LDFLAGS:-} -framework UniformTypeIdentifiers" go build -tags desktop,production -ldflags '-s -w' -o "$bundle/Contents/MacOS/go-peek" .
cp build/Info.plist "$bundle/Contents/Info.plist"
# A default ad hoc DR uses the changing executable hash, invalidating local
# privacy grants after rebuilds. Use an explicit identifier-only DR for local
# development, or a certificate-based DR when a signing identity is supplied.
bundle_id=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$bundle/Contents/Info.plist")
if [[ -n "${GOPEEK_SIGN_IDENTITY:-}" ]]; then
  codesign --force --sign "$GOPEEK_SIGN_IDENTITY" --identifier "$bundle_id" "$bundle"
else
  codesign --force --sign - --identifier "$bundle_id" \
    --requirements "=designated => identifier \"$bundle_id\"" "$bundle"
fi
codesign --verify --deep --strict "$bundle"
echo "Built $bundle (signed local development bundle)."
