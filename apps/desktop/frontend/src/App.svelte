<script lang="ts">
  import { onMount } from 'svelte';
  import { desktop, onEvent, type StreamEvent } from './bridge';
  let marker = $state('CADDY-CAPTURE-TEST');
  let output = $state('');
  let error = $state('');
  let active = $state('');
  let busy = $state(false);
  let pinned = $state(false);
  let requested = $state<boolean | undefined>();
  let early: StreamEvent[] = [];

  function apply(event: StreamEvent) {
    if (event.requestId !== active) return;
    if (event.kind === 'text') output += event.text ?? '';
    else busy = false;
  }
  onMount(() => {
    const stopStream = onEvent<StreamEvent>('demo:stream', event => {
      if (busy && !active) early.push(event); else apply(event);
    });
    if (window.go) {
      desktop().GetStatus().then(status => { requested = status.captureProtectionRequested; }).catch(e => { error = String(e); });
    }
    return stopStream;
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
    await control(() => desktop().Reset());
    active = ''; early = []; busy = false; output = ''; marker = 'CADDY-CAPTURE-TEST';
  }
</script>

<main>
  <header><span class="eyebrow">ASSESSMENT CADDY / PHASE 0</span><h1>Capture feasibility</h1><p>A compact desktop window for testing the workflow your first release depends on.</p></header>
  <section class="notice" aria-label="Capture protection status">
    <strong>Capture exclusion is unverified</strong>
    <p>Protection flag: {requested === undefined ? 'awaiting desktop status' : requested ? 'requested' : 'off — control run'}. A successful flag request does not prove this window is excluded from a recording.</p>
  </section>
  <section>
    <label for="marker">Visible test marker</label>
    <textarea id="marker" rows="3" bind:value={marker} disabled={busy} placeholder="Use a unique, nonsensitive marker" spellcheck="false"></textarea>
    <p class="help">Use test text only. This prototype makes no API requests.</p>
    <div class="actions"><button onclick={run} disabled={busy || !marker.trim()}>Stream demo</button><button class="secondary" onclick={() => control(() => desktop().Cancel())} disabled={!busy || !active}>Cancel</button><button class="secondary" onclick={reset} disabled={busy && !active}>Reset</button></div>
  </section>
  <section>
    <div class="section-heading"><h2>Demo response</h2><span>{busy ? 'Streaming…' : 'Idle'}</span></div>
    <pre aria-live="polite" aria-busy={busy}>{output || 'The demo stream will appear here.'}</pre>
    <p class="help">No model output. No tests executed. Content stays in memory until reset or quit.</p>
  </section>
  <section class="window-controls">
    <label class="checkbox"><input type="checkbox" bind:checked={pinned} onchange={() => control(() => desktop().SetAlwaysOnTop(pinned))}/> Always on top</label>
    <button class="secondary" onclick={() => control(() => desktop().HideTemporarily())}>Hide for 3 seconds</button>
  </section>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <footer>macOS prototype · Resize and move using the native window controls. Global shortcuts and live providers are pending.</footer>
</main>
