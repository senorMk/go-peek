<script lang="ts">
  import { onMount } from 'svelte';
  import AlwaysOnTop from './AlwaysOnTop.svelte';
  import { desktop, onEvent, type CaptureEvent, type CapturedScreenshot, type PrototypeStatus } from './bridge';
  let { images = $bindable<CapturedScreenshot[]>([]), disableDiscard = false, resetVersion = 0 }: { images?: CapturedScreenshot[]; disableDiscard?: boolean; resetVersion?: number } = $props();
  let capturing = $state(false);
  let nextID = 0;
  const maxImages = 20, maxPixels = 64_000_000, maxEncodedBytes = 112 * 1024 * 1024;
  function append(image: CaptureEvent['image']) {
    if (!image) return;
    const pixels = images.reduce((total, item) => total + item.width * item.height, image.width * image.height);
    const bytes = images.reduce((total, item) => total + item.dataUrl.length, image.dataUrl.length);
    if (images.length >= maxImages || pixels > maxPixels || bytes > maxEncodedBytes) {
      error = 'Screenshot list is full. Remove some screenshots before capturing more.'; message = ''; return;
    }
    images = [...images, { ...image, id: ++nextID }]; message = 'Screenshot added';
  }
  let message = $state(''), error = $state('');
  $effect(() => { if (resetVersion > 0) { message = ''; error = ''; } });
  let shortcutReady = $state(false), shortcutError = $state('');
  function status(value: PrototypeStatus) {
    shortcutReady = value.captureShortcutRegistered;
    shortcutError = value.captureShortcutError ?? '';
  }
  onMount(() => {
    const stop = onEvent<CaptureEvent>('capture:result', event => {
      if (event.kind === 'started') { capturing = true; error = ''; message = 'Capturing…'; return; }
      capturing = false;
      if (event.kind === 'done' && event.image) { append(event.image); }
      else if (event.kind === 'cancelled') message = 'Capture cancelled';
      else { error = event.error ?? 'Could not capture screenshot.'; message = ''; }
    });
    const stopStatus = onEvent<PrototypeStatus>('prototype:status', status);
    if (window.go) void desktop().GetStatus().then(status).catch(e => { shortcutError = String(e); });
    return () => { stop(); stopStatus(); images = []; };
  });
  async function capture(mode: 'screen' | 'region') {
    error = '';
    try { await desktop().CaptureScreenshot(mode); }
    catch (e) { error = String(e); }
  }
</script>

<section class="screenshot-panel" aria-label="Screenshot capture" aria-busy={capturing}>
  <div class="actions">
    <AlwaysOnTop/>
    <button class="secondary" disabled={capturing} onclick={() => capture('screen')}>Capture screen {shortcutReady ? '(⌘⇧S)' : ''}</button>
    <button class="secondary" disabled={capturing} onclick={() => capture('region')}>Capture region</button>
    {#if images.length}<button class="secondary" disabled={disableDiscard || capturing} onclick={() => { images = []; message = ''; }}>Clear screenshots</button>{/if}
  </div>
  <p class="help">Screen captures the main display. Region: drag the green dotted selection; release to capture. Esc cancels. GoPeek hides during capture.</p>
  {#if shortcutError}<p class="help">{shortcutError} Use the capture buttons.</p>{/if}
  <p class="help" role="status">{message}</p>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if images.length}
    <h2>Captured screenshots ({images.length})</h2>
    <ol class="screenshot-list">
      {#each images as image, index (image.id)}
        <li class="screenshot-item">
          <img class="screenshot-preview" src={image.dataUrl} alt={`Captured screenshot ${index + 1}`} />
          <div class="section-heading">
            <p class="help">Screenshot {index + 1} · {image.width} × {image.height}</p>
            <button class="secondary" disabled={disableDiscard || capturing} aria-label={`Remove screenshot ${index + 1}`} onclick={() => { images = images.filter(item => item.id !== image.id); }}>Remove</button>
          </div>
        </li>
      {/each}
    </ol>
    <p class="help">Screenshots stay local until you ask. Review the list before sending it to the selected provider.</p>
  {/if}
</section>
