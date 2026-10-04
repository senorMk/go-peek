# Assessment Caddy

A lightweight desktop coding assistant in development. The first implementation is a **macOS capture-feasibility prototype**, using Go, Wails, Svelte, and TypeScript. It does not yet call OpenAI or OpenCode Zen.

Capture exclusion is a release requirement. Phase 0 remains open until recordings from the required capture tools demonstrate exclusion. The prototype explicitly labels the protection flag as unverified.

## Run the prototype

Requires macOS, Xcode Command Line Tools, Go 1.26+, and Node 22.12+.

```sh
npm ci
make check
make test
make prototype
open "apps/desktop/build/bin/Assessment Caddy.app"
```

The local bundle is signed ad hoc for development; it is not Developer ID signed or notarized for distribution. `make prototype` builds the frontend and embeds it in the executable; no frontend dev server is required to run the bundle. There are no provider calls, API keys, remote fonts, or telemetry in this prototype.

Use a nonsensitive, unique test marker. Stream the demo, cancel it, reset it, move/resize the window, toggle always-on-top, and hide it for three seconds. Closing the window quits the app. Output is plain text, kept in memory, and never executed.

Run an unprotected control from Terminal after quitting the protected instance:

```sh
CADDY_CAPTURE_PROTECTION=off "apps/desktop/build/bin/Assessment Caddy.app/Contents/MacOS/assessment-caddy"
```

Compare both variants using the [capture test procedure](docs/FEASIBILITY.md). The flag defaults to enabled; the UI reports what was requested, not whether capture is prevented.

`npm run preview` starts a browser preview of the layout. Native controls require the desktop app. `make build` produces a standalone executable; `make prototype` produces the macOS app bundle. A globally installed Wails CLI is not required. For Wails development tooling, use the same pinned version as `apps/desktop/go.mod`.

## Layout

```text
apps/desktop/
  main.go                 Wails lifecycle and narrow frontend bridge
  internal/session/       In-memory demo stream, identities, cancellation/reset
  frontend/               Svelte/TypeScript window UI
  scripts/                Local macOS packaging
  build/Info.plist         Prototype bundle metadata
docs/FEASIBILITY.md        Decisions, compatibility matrix, and measurement procedure
PLAN.md                    Full product plan and implementation status
```

The [Go monorepo template](https://github.com/senorMk/go-monorepo-template) inspired the app-per-directory structure, root commands, lockfiles, and CI checks. The desktop app follows the plan's local-only architecture. Core session code has no Wails dependency and can be tested headlessly.

## Validation

`make check` checks Svelte/TypeScript and vets Go. `make test` runs Go tests with the race detector. CI also packages the macOS app. Session tests cover validation, competing requests, cancellation, reset isolation, request identities, and normal completion. These checks do not establish capture exclusion or validate native window behavior.

## Next gate

Confirm the target macOS versions and recording/sharing applications, record protected and control runs, and inspect the results. Then resolve the macOS capture-exclusion requirement before committing to the full UI. Global shortcut registration, performance measurements, native lifecycle checks, secure key storage, provider adapters, and the five coding-assistance actions are still pending. First-release language choices are Go, Python, and JavaScript/TypeScript; the second provider is OpenCode Zen.
