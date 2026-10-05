<script lang="ts">
  import { onMount } from 'svelte';
  import AlwaysOnTop from './AlwaysOnTop.svelte';
  import { desktop, onEvent, type StreamEvent, type PrototypeStatus } from './bridge';
  let marker = $state('GOPEEK-CAPTURE-TEST');
  let output = $state('');
  let error = $state('');
  let active = $state('');
  let busy = $state(false);
  let requested = $state<boolean | undefined>();
  let shortcut = $state('⌘B');
  let shortcutRegistered = $state(false);
  let shortcutError = $state('');
  let early: StreamEvent[] = [];
  function updateStatus(status: PrototypeStatus) {
    requested = status.captureProtectionRequested;
    shortcut = status.shortcut;
    shortcutRegistered = status.shortcutRegistered;
    shortcutError = status.shortcutError ?? '';
  }

  function apply(event: StreamEvent) {
    if (event.requestId !== active) return;
    if (event.kind === 'text') output += event.text ?? '';
    else busy = false;
  }
  onMount(() => {
    const stopStream = onEvent<StreamEvent>('demo:stream', event => {
      if (busy && !active) early.push(event); else apply(event);
    });
    const stopStatus = onEvent<PrototypeStatus>('prototype:status', updateStatus);
    if (window.go) {
      desktop().GetStatus().then(updateStatus).catch(e => { error = String(e); });
    }
    return () => { stopStream(); stopStatus(); };
  });
  async function run() {
    error = ''; output = ''; active = ''; early = []; busy = true;
    try {
      active = await desktop().StartDemo(marker);
      early.forEach(apply); early = [];
    } catch (e) { busy = false; error = String(e); }
  }
  async function control(action: () => Promise<void>) {
    error = '';
    try { await action(); } catch (e) { error = String(e); }
  }
  async function reset() {
    await control(() => desktop().ResetDemo());
    active = ''; early = []; busy = false; output = ''; marker = 'GOPEEK-CAPTURE-TEST';
  }
</script>

<main>
  <header><span class="eyebrow">GOPEEK / PHASE 0</span><h1>Capture feasibility</h1><p>A compact desktop window for testing the workflow your first release depends on.</p></header>
  <section class="notice" aria-label="Capture protection status">
    <strong>Capture exclusion is unverified</strong>
    <p>Protection flag: {requested === undefined ? 'awaiting desktop status' : requested ? 'requested' : 'off — control run'}. A successful flag request does not prove this window is excluded from a recording.</p>
  </section>
  <section>
    <label for="marker">Visible test marker</label>
    <textarea id="marker" rows="3" bind:value={marker} disabled={busy} placeholder="Use a unique, nonsensitive marker" spellcheck="false"></textarea>
    <p class="help">Use test text only. This prototype makes no API requests.</p>
    <div class="actions"><button onclick={run} disabled={busy || !marker.trim()}>Stream demo</button><button class="secondary" onclick={() => control(() => desktop().CancelDemo())} disabled={!busy || !active}>Cancel</button><button class="secondary" onclick={reset} disabled={busy && !active}>Reset</button></div>
  </section>
  <section>
    <div class="section-heading"><h2>Demo response</h2><span>{busy ? 'Streaming…' : 'Idle'}</span></div>
    <pre aria-live="polite" aria-busy={busy}>{output || 'The demo stream will appear here.'}</pre>
    <p class="help">No model output. No tests executed. Content stays in memory until reset or quit.</p>
  </section>
  <section class="window-controls">
    <AlwaysOnTop/>
    <button class="secondary" onclick={() => control(() => desktop().HideTemporarily())}>Hide for 3 seconds</button>
  </section>
  <p class="help">Show/hide shortcut: <kbd>{shortcut}</kbd> — {shortcutRegistered ? 'registered; test while another app has focus' : shortcutError ? 'unavailable' : 'waiting for registration'}.</p>
  {#if shortcutError}<p class="error" role="alert">{shortcutError}</p>{/if}
  <details>
    <summary>Capture test steps</summary>
    <ol>
      <li>Use a unique marker and record the full display in OBS with its macOS Screen Capture source.</li>
      <li>Stream the demo, move/resize this window, switch focus, and test the shortcut and timed hide.</li>
      <li>Inspect the recording for the marker and window during every transition.</li>
      <li>Quit, run the protection-off control described in README, and repeat. The control marker must appear for the comparison to be valid.</li>
    </ol>
    <p class="help">A successful shortcut registration or an absent marker in a selected-window share does not establish display-wide capture exclusion.</p>
  </details>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <footer>macOS prototype · Resize and move using the native window controls. Capture compatibility and live providers are pending.</footer>
</main>
