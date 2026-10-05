#!/usr/bin/env bash
# Run from an interactive macOS Terminal, with existing GoPeek instances closed.
set -euo pipefail
umask 077
repo_dir="$(cd "$(dirname "$0")/../../.." && pwd)"
binary="$repo_dir/apps/desktop/build/bin/GoPeek.app/Contents/MacOS/go-peek"
mode="${1:-screenshot}"
share_mode="${2:-display}"
external=false
case "$mode" in
  screenshot) extension=png ;;
  video) extension=mov ;;
  meet|obs|zoom)
    external=true
    case "$share_mode" in
      display|window) ;;
      *) echo "External capture mode must be display or window." >&2; exit 2 ;;
    esac
    ;;
  *) echo "Usage: bash apps/desktop/scripts/capture-check.sh [screenshot|video|meet|obs|zoom] [display|window]" >&2; exit 2 ;;
esac
if [[ "$(uname -s)" != Darwin ]]; then
  echo "This check requires an interactive macOS desktop." >&2
  exit 1
fi
if [[ ! -x "$binary" ]]; then
  echo "Build the bundle first: make prototype" >&2
  exit 1
fi
if [[ ! -t 0 ]]; then
  echo "Run this check from an interactive Terminal so you can position the test window." >&2
  exit 1
fi
mkdir -p "$repo_dir/.cache/capture-checks"
result_dir="$(mktemp -d "$repo_dir/.cache/capture-checks/run-XXXXXX")"
app_pid=""
stop_app() {
  if [[ -n "$app_pid" ]]; then
    kill "$app_pid" 2>/dev/null || true
    wait "$app_pid" 2>/dev/null || true
    app_pid=""
  fi
}
trap stop_app EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

tool_description='/usr/sbin/screencapture (built-in; no audio requested)'
if [[ "$external" == true ]]; then
  printf 'Testing %s, %s sharing/capture. Open the tool yourself; no call is started by this script.\n' "$mode" "$share_mode"
  read -r -p 'Enter the tool version (for Meet include the Chrome version): ' tool_description
  if [[ -z "$tool_description" ]]; then echo "A tool version is required for a reproducible result." >&2; exit 2; fi
  tool_description="$mode: $tool_description"
fi
report="$result_dir/RESULTS.md"
{
  printf '# Capture comparison\n\n'
  if [[ "$external" == true ]]; then
    printf 'Mode: %s / %s.\n\n' "$mode" "$share_mode"
  else
    printf 'Mode: %s on display 1 (main display).\n\n' "$mode"
  fi
  printf 'Date (UTC): %s\n\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  printf 'macOS: %s (%s); architecture: %s\n\n' "$(sw_vers -productVersion)" "$(sw_vers -buildVersion)" "$(uname -m)"
  printf 'Working tree revision: %s\n\n' "$(git -C "$repo_dir" describe --always --dirty 2>/dev/null || printf 'unavailable')"
  printf 'Executable SHA-256: %s\n\n' "$(shasum -a 256 "$binary" | awk '{print $1}')"
  printf 'Tool: %s\n\n' "$tool_description"
  printf '| Run | Capture command | Marker/window visible? |\n| --- | --- | --- |\n'
} > "$report"

printf 'Close all existing GoPeek instances before continuing. Captures stay in %s\n' "$result_dir"
read -r -p 'Press Return once GoPeek is closed and the main display is ready for capture: '
for variant in protected control; do
  flag=on
  expected_status=requested
  if [[ "$variant" == control ]]; then flag=off; expected_status=off; fi
  env GOPEEK_CAPTURE_PROTECTION="$flag" "$binary" > "$result_dir/$variant-launch.log" 2>&1 &
  app_pid=$!
  printf '\n%s run: position GoPeek visibly on the main display. Confirm the protection flag says %s.\n' "$variant" "$expected_status"
  printf 'Enable Show Capture checks in Settings → Diagnostics, then open the Capture checks tab. Use marker GOPEEK-CAPTURE-TEST. Exercise streaming, move/resize, focus changes, always-on-top, and Command+B.\n'
  if [[ "$external" == true ]]; then
    if [[ "$share_mode" == display ]]; then
      printf 'Share/capture the entire main display, with GoPeek visibly overlapping the editor.\n'
    else
      printf 'Choose the GoPeek window itself. Sharing only the editor is a separate workflow and does not test GoPeek exclusion.\n'
    fi
    if [[ "$mode" == obs ]]; then
      printf 'Use the macOS Screen Capture source and its Display or Window Capture method. Inspect the saved recording.\n'
    else
      printf 'Inspect the receiver view on your own second device/window, or a saved recording; the local preview alone is insufficient.\n'
    fi
  else
    printf 'For video, exercise the transitions during the 10-second recording.\n'
  fi
  read -r -p 'Press Return when the test window is visible and ready: '
  if ! kill -0 "$app_pid" 2>/dev/null; then
    printf '| %s | App exited before capture; inspect %s-launch.log | Inconclusive |\n' "$variant" "$variant" >> "$report"
    echo "The app exited. Capture comparison is inconclusive; see $report" >&2
    exit 1
  fi
  if [[ "$external" == true ]]; then
    printf 'Complete and inspect the %s capture before entering an observation.\n' "$variant"
    printf 'Record absent only when GoPeek was visible locally during the compared capture; manually hiding it is not exclusion.\n'
    read -r -p 'GoPeek in the captured output: visible / absent / not-selectable / inconclusive: ' observation
    case "$observation" in
      visible|absent|inconclusive) ;;
      not-selectable)
        if [[ "$share_mode" != window ]]; then echo "not-selectable applies only to window capture." >&2; exit 2; fi
        ;;
      *) echo "Invalid observation; this run is incomplete." >&2; exit 2 ;;
    esac
    read -r -p 'Local evidence file path (leave blank if unavailable): ' evidence_file
    evidence_status='No artifact supplied; user observation only'
    if [[ -n "$evidence_file" ]]; then
      if [[ ! -f "$evidence_file" || ! -s "$evidence_file" ]]; then echo "Evidence must be an existing nonempty local file." >&2; exit 2; fi
      evidence_name="${evidence_file##*/}"
      cp "$evidence_file" "$result_dir/$variant-$evidence_name"
      evidence_status="Local evidence: $variant-$evidence_name"
    fi
    printf '| %s | %s | %s (user observation) |\n' "$variant" "$evidence_status" "$observation" >> "$report"
    stop_app
    continue
  fi
  file="$result_dir/$variant.$extension"
  if [[ "$mode" == video ]]; then
    capture_args=(-x -D 1 -v -V 10 "$file")
  else
    capture_args=(-x -D 1 "$file")
  fi
  if /usr/sbin/screencapture "${capture_args[@]}" > "$result_dir/$variant-capture.log" 2>&1 && [[ -s "$file" ]]; then
    printf '| %s | Saved %s.%s | Pending inspection |\n' "$variant" "$variant" "$extension" >> "$report"
  else
    printf '| %s | Capture failed; inspect %s-capture.log | Inconclusive |\n' "$variant" "$variant" >> "$report"
    echo "Capture failed. Check Terminal screen-recording permission and the capture log; rerun after resolving it." >&2
    exit 1
  fi
  stop_app
done
{
  printf '\n## Inspection\n\n'
  printf -- '- [ ] Control marker and window are visible. If absent, the comparison is inconclusive.\n'
  printf -- '- [ ] Protected marker/window result recorded (visible = failure for this mode; absent = exclusion observed for this run).\n'
  printf -- '- [ ] For video, inspect every transition; hiding the app manually is not capture exclusion.\n'
  printf -- '- [ ] Record observations and evidence paths in this RESULTS.md.\n\n'
  printf 'These results cover only the recorded tool, version, and selected mode; they do not establish universal capture exclusion. External observations are user-reported and require evidence review before acceptance.\n'
} >> "$report"
printf '\nCaptures and inspection checklist saved to %s\n' "$result_dir"
