# Phase 0 — macOS feasibility

## Confirmed scope

- First platform: macOS.
- Capture exclusion: essential; Phase 0 is a gate for full UI development.
- OpenCode credential: OpenCode Zen.
- Initial languages: Go, Python, JavaScript/TypeScript.
- Initial input: pasted text is planned; screenshots follow the text workflow. Launch input requirements still need confirmation.
- Required capture applications and supported macOS versions: awaiting confirmation.

## Recommended initial test scope

Start with the current macOS 26 development host. The proposed first matrix is built-in screenshots/recordings, Google Meet in Chrome, Zoom desktop, and OBS. This is a recommendation, not confirmed user usage or a tested support promise. Add Teams when it is a required workflow. After current-host behavior passes, add macOS 15 and other versions required by actual users; do not promise compatibility based on the bundle minimum version.

Test OBS first with its **macOS Screen Capture** source in Display Capture and Window Capture modes. [OBS documents that this source uses ScreenCaptureKit](https://obsproject.com/kb/macos-screen-capture-source), making it a direct check of the known legacy-flag limitation. In Meet and Zoom test both full-display sharing and selected-window sharing, and inspect the receiver's output. [Zoom documents multiple capture settings](https://support.zoom.com/hc/en/article?id=zm_kb&sysparm_article=KB0063824); record the selected capture mode and avoid assuming results transfer between settings.

## Framework candidate

Wails **v2.15.0** is pinned for this experiment, not a final framework decision. [Wails documentation](https://v2.wails.io/docs/introduction/) identifies v2 as stable; [v3 status](https://v3.wails.io/status/) identifies v3 as beta. This first spike uses the stable framework to inspect the capture path before introducing beta desktop APIs.

The prototype uses a native movable/resizable window, optional always-on-top, timed hide/restore, and a deterministic cancellable stream. Native dialogs and additional windows are absent. Global shortcuts are pending; timed restore is not a substitute for validating them.

### Capture implementation finding

Inspection of Wails v2.15.0's `internal/frontend/desktop/darwin/Application.m` shows `mac.Options.ContentProtection` calls `setSharingType:NSWindowSharingNone` during main-window creation. The [pinned source](https://github.com/wailsapp/wails/blob/v2.15.0/internal/frontend/desktop/darwin/Application.m) is the implementation reference.

[Electron's documentation](https://www.electronjs.org/docs/latest/api/browser-window#winsetcontentprotectionenable) describes the same underlying macOS API and warns that ScreenCaptureKit capture can include protected windows. This is evidence that the legacy flag cannot establish the required compatibility; it is not a recorded result for this app. No repeated flag application or process disguise has been added.

The setting is enabled by default. `CADDY_CAPTURE_PROTECTION=off` creates an otherwise identical control run. The UI displays the requested flag and always states exclusion is unverified.

**Decision status: OPEN.** Do not describe the application as capture-excluded or move to the full UI until recordings establish acceptable behavior. If required ScreenCaptureKit tools capture the protected window, record failure and decide whether another supported native approach, another platform, or a revised workflow is acceptable. Switching Wails major versions alone is not evidence of a fix.

## Provider protocol findings

[OpenCode Zen's endpoint table](https://opencode.ai/docs/zen/#endpoints) documents separate Responses, Chat Completions, Messages, and other model-specific endpoints. For a later narrow first adapter, its GPT-family models use `https://opencode.ai/zen/v1/responses`; choose and reverify exact models when implementing the adapter. Do not infer that every Zen model supports that protocol. No authenticated request has been made and no key has been requested or read.

OpenAI SDK/Responses integration will be researched when that adapter is implemented. The demo does not exercise either provider.

## Compatibility matrix

Record app commit/build identity, macOS version/build, hardware, capture tool/version, mode, flag, marker, recording path, and observed result for every run. This workspace's development host reports macOS 26.6.2 on Apple Silicon; minimum supported versions are not established by a successful build. The bundle's macOS 12 minimum is a provisional packaging setting, not a tested support claim.

| macOS | Tool/version | Capture mode | Protected | Control | Recording evidence | Status |
| --- | --- | --- | --- | --- | --- | --- |
| 26.6.2 (development host) | Built-in screenshot | Display / selected region | Pending | Pending | None | Unverified |
| 26.6.2 (development host) | Built-in recording | Full display / selected region | Pending | Pending | None | Unverified |
| 26.6.2 (proposed first matrix) | OBS, version pending | ScreenCaptureKit display / window | Pending | Pending | None | Unverified |
| 26.6.2 (proposed first matrix) | Chrome + Google Meet, versions pending | Full display / selected window | Pending | Pending | None | Unverified |
| 26.6.2 (proposed first matrix) | Zoom desktop, version pending | Full display / selected window | Pending | Pending | None | Unverified |

A passing result is a recording where the control marker is visible and the protected marker is absent through the entire tested lifecycle. If the control is absent too, the run does not validate protection. A capture error is inconclusive, not a pass.

## Manual capture procedure

1. Build with `make prototype`. Quit previous instances. Open the protected bundle, verify its status says requested, and use a unique nonsensitive marker.
2. Start recording/sharing in the required tool. For shared output, inspect the receiver's view or saved recording.
3. Exercise initial creation, moving/resizing, switching focus, always-on-top on/off, streaming, cancellation/reset, and timed hide/restore. Quit and reopen while recording to cover window recreation.
4. Stop and inspect the captured output frame by frame around transitions. Record whether the app, title bar, marker, and response appeared; do not infer success from the local display or the flag alone.
5. Quit the app and run the unprotected control using the command in README. Repeat the same capture modes and lifecycle.
6. Repeat full-display, app-window, screenshot, selected-region, and recording modes. A selected coding-window share is a separate workflow; it does not prove display-wide exclusion.
7. Record results and evidence paths in the matrix. Any future auxiliary/settings window must have its own tests before being called protected.

Permission prompts from the recording tool are part of its workflow. This app does not request screen capture permission or capture the display in this spike.

## Performance baseline procedure

Use the same release bundle and hardware for every measurement. Measure cold launch to first rendered frame, idle CPU and resident memory after 30 seconds, peak memory while streaming, and recovery after reset. Measure both protection variants. Capture test tooling overhead separately.

| Metric | Result | Measurement method |
| --- | --- | --- |
| Bundle size | 7,572 KiB allocated (~7.4 MiB), ad hoc signed arm64 bundle | `du -sk "apps/desktop/build/bin/Assessment Caddy.app"` |
| Startup to first frame | Pending manual measurement | Repeated cold launches; record median and range |
| Idle CPU / RSS | Pending manual measurement | Activity Monitor or Instruments, 30-second settled window |
| Streaming peak RSS | Pending manual measurement | Same tool and workload across runs |

## Phase 0 exit checklist

- [x] Platform, key service, and initial language choices confirmed.
- [x] Desktop feasibility source and cancellation/reset checks implemented.
- [x] macOS framework protection implementation inspected.
- [x] Zen protocol differences checked in official documentation.
- [x] Native arm64 app bundle built; Svelte check, frontend build, Go vet, and Go race tests pass.
- [ ] Global shortcut registration and lifecycle validated.
- [ ] Required OS versions, input methods, and capture tools confirmed.
- [ ] Protected/control recordings inspected and compatibility results accepted.
- [ ] Startup, idle, streaming, and packaged size baseline measured.
- [ ] Framework/version finalized after the capture gate.

## Automated verification record — 2026-10-04

- Svelte/TypeScript: zero errors and warnings.
- Vite: production frontend built and embedded.
- Go: vet and race-enabled tests passed.
- macOS: arm64 release executable and app bundle built; Info.plist validation and strict bundle signature verification passed.
- The direct Go packaging path explicitly links UniformTypeIdentifiers, which the selected Wails source uses with the current SDK. The completed bundle is signed ad hoc, matching the Wails packaging behavior; the Go linker signature alone was insufficient after adding bundle metadata. An upstream NSToolbar deprecation warning remains; the build succeeds.
- A native launch smoke check in this terminal environment aborted with SIGABRT inside Wails' macOS window creation. Launch Services also returned `kLSNoExecutableErr` despite the executable existing and strict bundle signature verification passing. Both launch checks still failed after ad hoc signing. The cause has not been established; this is a failed launch check, not native validation. Launching the bundle from the interactive desktop and inspecting any crash report is the next check.
- Native rendering, window controls, capture behavior, and performance beyond disk size still require manual checks. No display contents have been captured or sent to a provider.
