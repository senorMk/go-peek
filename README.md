# GoPeek

**Capture a problem. Find your next step.**

A lightweight desktop coding assistant in development. The current build provides a **macOS screenshot-first coding workflow** using Go, Wails, Svelte, and TypeScript: hints, approach explanations, code reviews, debugging, full solutions, streaming, and follow-ups. OpenAI and OpenCode Zen adapters are implemented and covered by mock HTTP tests; live requests have not yet been verified.

Capture exclusion remains a release requirement. The built-in screenshot/video comparisons passed for the earlier tested bundle on macOS 26.6.2; additional tool/lifecycle checks are pending. At the user’s request, development continues while those checks are deferred. The capture UI keeps the flag status separate from validated compatibility.

## Run the prototype

Requires macOS, Xcode Command Line Tools, Go 1.26+, and Node 22.12+.

```sh
npm ci
make check
make test
make prototype
open "apps/desktop/build/bin/GoPeek.app"
```

The window starts at 520 × 620, is resizable down to 420 × 360, and uses a transparent macOS title bar over the dark app background. The native window buttons remain available; drag the top strip to move GoPeek. Optional problem details, examples, and current code are collapsed under **Add details, examples, or code**. Token usage uses thousands separators.

The local bundle is signed ad hoc with an explicit identifier-based designated requirement for development; it is not Developer ID signed or notarized for distribution. This keeps the local signing requirement stable across rebuilds rather than using a changing executable hash. For certificate-based signing, set `GOPEEK_SIGN_IDENTITY` when packaging. `make prototype` builds the frontend and embeds it in the executable; no frontend dev server is required to run the bundle. API calls happen only when you explicitly request a response in Work. There are no remote fonts, telemetry, or continuous screen capture. The separate Capture checks tab remains a deterministic demo with no API calls.

## Set up a provider and work through a problem

Open **Settings → Provider**, choose and save the provider/model, enter your key in the masked **API key** field, and click **Save API key**. The field clears after saving. The UI shows only whether a key is saved; it never retrieves or displays the saved value. To replace a key, enter a new one; **Delete saved key** removes it from Keychain.

The key is stored under GoPeek's own macOS Keychain service. A newly entered key passes through the native bridge only for storage; it is not written to settings or logs. Native Keychain save/access/deletion still needs desktop verification; automated tests do not touch your Keychain.

Terminal setup remains available as an alternative:

```sh
make setup-key PROVIDER=openai
# Or: make setup-key PROVIDER=zen
```

The Terminal prompt hides the key and passes it to the native executable over stdin, without putting it in command-line arguments.

Open **Settings → Provider**, select the provider/model, save, and refresh key status. OpenAI accepts an explicitly configured Responses-compatible model ID. Zen currently allows `gpt-6-luna`, `gpt-6-sol`, and `gpt-6.1-sol` through its documented Responses endpoint; this is a protocol subset, not a live-tested compatibility claim. No cross-provider fallback occurs. You can delete the selected provider's saved key from this panel.

Capture one or more screenshots of the problem. The Work workspace then opens; typed problem details, examples, code, and failing-test information are optional additions. Choose from 32 programming and scripting languages and one of the five actions. The alphabetized selector includes Python, JavaScript, TypeScript, Java, C, C++, C#, Go, Rust, Kotlin, Swift, PHP, Ruby, Dart, SQL, and more. Keep **Include screenshots** checked and ask for a response directly from the images. A saved provider key is required; a typed statement is not. You can explicitly uncheck images for a text-only request. Cancel when needed, ask follow-ups after a complete response, and choose **Clear session** to clear inputs, screenshots, and context. Saving provider/model settings also resets conversation context.

Keys stay in Keychain; non-secret provider/model and Always on top settings live in `~/Library/Application Support/go-peek/settings.json`. Problem/code and conversations stay in memory. Generation uses the official OpenAI Go SDK with the selected provider's explicit endpoint, `store=false`, disabled automatic truncation, no automatic retries, a three-minute request deadline, and a 4,096 output-token limit. Provider retention policies are separate from the storage flag. Partial/cancelled responses remain visible but are not added to follow-up context.

Input fields have a 24 KiB total cap; assembled text conversation context has a 64 KiB cap; screenshot requests have a separate 32 MiB total PNG cap, with at most 20 images and 64 total megapixels; response display has a 32 KiB cap. These are conservative byte limits, not a model-specific tokenizer. A model may impose stricter limits; errors are displayed and nothing is silently truncated. Model output is rendered as escaped text with bold emphasis, inline code, and fenced code, with no raw HTML or automatic link/code execution. Full rich Markdown formatting is pending.

Screenshot requests are enabled for the explicit `gpt-6-luna`, `gpt-6-sol`, and `gpt-6.1-sol` subset. Other configured models retain text-only input until their image capability is added; no model or provider is switched automatically. Image content remains separate from the text prompt and is retained in memory for completed-request follow-ups until reset. New captures are included on the next main request; follow-ups retain the screenshots from the last completed request. Removing a preview does not erase that conversation; choose **Clear session** to clear it.

The request format follows [OpenAI’s image input documentation](https://developers.openai.com/api/docs/guides/images-vision) and its [GPT-6 Luna](https://developers.openai.com/api/docs/models/gpt-6-luna), [GPT-6 Sol](https://developers.openai.com/api/docs/models/gpt-6-sol), and [GPT-6.1 Sol](https://developers.openai.com/api/docs/models/gpt-6.1-sol) model documentation. Zen’s Responses compatibility uses its documented endpoint; successful live image requests remain unverified.

Provider sources: [official OpenAI streaming documentation](https://developers.openai.com/api/docs/guides/streaming-responses), [OpenCode Zen endpoints](https://opencode.ai/docs/zen/#endpoints).

## Quick screenshots

Work opens with capture controls and a prompt to capture a problem. **Work through the problem** appears after a successful screenshot. Each successful capture appends to the ordered screenshot list. Removing the last screenshot or choosing **Clear screenshots** hides that workspace; **Clear session** clears the list and conversation. Cancelled or failed captures leave the list intact. **Clear session** is always available in the top navigation and returns to the capture prompt, cancelling generation and clearing all question inputs, previews, output, usage, and follow-up context. Provider settings, saved keys, language selection, and Always on top are retained. The screenshot list is sent in order only when you ask with **Include screenshots** checked. The assistant reads the problem from the images; typed details are optional.

The **Always on top** toggle beside the capture buttons keeps GoPeek above other windows. Its state is shared with **Settings → Window** and Capture checks. Changes save immediately to the local settings file and are applied when the native window is created on the next launch. Existing settings files without this preference default to off. Failed saves show an error and leave the current window preference unchanged; provider/model edits preserve the current window preference.

Use **Capture screen** or **⌘⇧S** while GoPeek is running to capture the main display, even when another app has focus. Use **Capture region** for the custom selector: drag a green dashed rectangle on a display and release to capture, or press Esc to cancel. The selector dims the surrounding screen, clamps the rectangle to the display where the drag began, and closes before taking the screenshot. Switching apps or changing the display layout cancels selection. **⌘B** remains the show/hide shortcut. GoPeek hides during capture and returns with the preview. If shortcut registration fails, the buttons remain available.

Screenshots remain local until you ask; images included in a request are sent to your selected provider as PNG image inputs. Captures use a private temporary directory, deleted after reading, and previews stay in memory until removed or the app closes. Use **Remove** on an individual screenshot or **Clear screenshots** to empty the list. Individual captures are capped at 20 MiB and 64 megapixels. The list is capped at 20 images, 64 total megapixels, and 112 MiB of encoded image data; excess captures are rejected while existing images remain. Region selection times out after two minutes. Capture and overlay controls prevent overlapping captures and keep GoPeek hidden until completion.

GoPeek checks and requests its own screen recording permission on a capture action, before hiding the window. If macOS does not recognize the grant, quit GoPeek, remove the old app entry from **System Settings → Privacy & Security → Screen & System Audio Recording**, add the current `apps/desktop/build/bin/GoPeek.app`, and reopen it. After the rename, macOS may require a new grant for GoPeek’s bundle identifier. Other screenshot utility failures report a separate category or exit status without exposing command stderr. See [Apple’s screen recording permission guide](https://support.apple.com/guide/mac-help/control-access-screen-system-audio-recording-mchld6aa7d23/mac). Native permission recovery, region cancellation, multiple displays, and the new global shortcut still need desktop verification; automated tests use generated PNG fixtures, never your screen.

## Capture feasibility

Capture checks is hidden by default. Enable **Settings → Diagnostics → Show Capture checks**, then open **Capture checks** for the retained demo. This choice saves immediately and persists across restarts. Use a nonsensitive, unique test marker. Stream the demo, cancel it, reset it, move/resize the window, toggle always-on-top, and hide it for three seconds. **⌘B** toggles visibility globally on key release; check its registration status in the app and test while another application has focus. Registration errors retain the timed-hide fallback. Closing the window quits the app. Output is plain text, kept in memory, and never executed.

Run an unprotected control from Terminal after quitting the protected instance:

```sh
GOPEEK_CAPTURE_PROTECTION=off "apps/desktop/build/bin/GoPeek.app/Contents/MacOS/go-peek"
```

For a guided built-in comparison, close GoPeek and run either command from your normal macOS Terminal:

```sh
make capture-check  # protected/control screenshots of the main display
make capture-video  # protected/control 10-second recordings, without audio
```

The runner launches each variant, waits for you to position the window, saves captures and build metadata under `.cache/capture-checks/`, and terminates only the instances it launched. Inspect both outputs and fill in the generated `RESULTS.md`; capture command success alone is not a pass. A failed launch or capture makes the run inconclusive. Recordings may require Terminal screen-recording permission. Capture files are ignored by Git and are not uploaded.

For Meet, OBS, or Zoom comparisons, run the external-tool guide:

```sh
make capture-external TOOL=meet MODE=display
# Then repeat with MODE=window, or TOOL=obs / TOOL=zoom.
```

Open the selected tool yourself and enter its version. The guide launches each GoPeek variant, asks for the receiver/recording observation, and optionally copies a local evidence file into the result folder. It starts no calls, invites no participants, and uploads nothing. For window mode, select GoPeek itself; selecting only the editor does not test capture exclusion.

Compare protected and unprotected captures, confirming the app was visible locally during each run. The flag defaults to enabled; the UI reports what was requested, not whether capture is prevented.

`npm run preview` starts a browser preview of the layout. Native controls require the desktop app. `make build` produces a standalone executable; `make prototype` produces the macOS app bundle. A globally installed Wails CLI is not required. For Wails development tooling, use the same pinned version as `apps/desktop/go.mod`.

## Layout

```text
apps/desktop/
  main.go                 Wails lifecycle and narrow frontend bridge
  internal/session/       In-memory demo stream, identities, cancellation/reset
  internal/overlay/       Visibility and timer coordination
  internal/platform/      macOS native global shortcut registration
  internal/capture/       Explicit local screenshots and private-file cleanup
  frontend/               Svelte/TypeScript window UI
  scripts/                Local macOS packaging
  build/Info.plist         Prototype bundle metadata
PLAN.md                    Full product plan and implementation status
```

The [Go monorepo template](https://github.com/senorMk/go-monorepo-template) inspired the app-per-directory structure, root commands, lockfiles, and CI checks. The desktop app follows the plan's local-only architecture. Core session code has no Wails dependency and can be tested headlessly.

## Validation

`make check` checks Svelte/TypeScript and vets Go. `make test` runs Go tests with the race detector. CI also packages the macOS app. Session tests cover validation, competing requests, cancellation, reset isolation, request identities, and normal completion. Live-session tests cover follow-up history, late events after reset, and bounded output/context; provider tests exercise the official SDK through in-memory HTTP transports for partial/malformed streams, errors, and deadlines. Settings tests verify private file permissions, validation, secret-field rejection, legacy-file compatibility, and Always on top persistence across reloads. Desktop bridge tests verify provider edits preserve the window preference and failed saves do not change its state. Overlay tests cover timed restoration, shortcut/timer interactions, capture visibility coordination, and shutdown. Screenshot tests cover native command arguments, fractional/negative desktop region coordinates, invalid rectangles, cancellation before capture, private directory permissions, PNG validation, sanitized failure categories, and temporary-file cleanup. Mock provider tests verify ordered multi-image payloads, explicit endpoints, image detail, storage flags, and rejection of unrecognized image models before any network request. Session tests verify screenshot-only prompts above the text byte limit, image snapshots, follow-up retention, and reset. Real shortcut registration and background key delivery require desktop testing. These checks do not establish capture exclusion or validate native window behavior.

## Remaining validation

Live OpenAI/Zen requests and native credential setup/access/deletion are pending. No normal test spends API credits. Capture testing across OBS, Meet, Zoom, all window transitions, and the rebuilt UI is deferred at the user's request. Signing/notarization for distribution, performance measurements, live screenshot-to-model verification, browser extraction, and Windows support remain later work. Phase 0 and Phase 1 are not marked complete.

The shared language catalog is [languages.json](apps/desktop/internal/prompts/languages.json); both the UI selector and Go validation use it. Language coverage was expanded using [GitHub Octoverse](https://github.blog/news-insights/octoverse/octoverse-a-new-developer-joins-github-every-second-as-ai-leads-typescript-to-1/) and the [Stack Overflow developer survey](https://survey.stackoverflow.co/2025/technology) as references, alongside additional languages used in coding practice. Selection steers the assistant’s response; the app does not execute these languages.
