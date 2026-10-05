#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "$0")/../../.." && pwd)"
provider="${1:-}"
case "$provider" in openai|zen) ;; *) echo "Usage: make setup-key PROVIDER=openai|zen" >&2; exit 2 ;; esac
binary="$repo_dir/apps/desktop/build/bin/GoPeek.app/Contents/MacOS/go-peek"
if [[ ! -x "$binary" ]]; then echo "Run make prototype first." >&2; exit 1; fi
if [[ ! -t 0 ]]; then echo "Run key setup in your normal interactive Terminal." >&2; exit 1; fi
gopeek_key=''
trap 'unset gopeek_key' EXIT
read -r -s -p "Enter your $provider API key (hidden): " gopeek_key
printf '\n'
# The key is piped to the native backend; it is never a process argument or a file.
printf '%s' "$gopeek_key" | "$binary" --store-key "$provider"
unset gopeek_key
