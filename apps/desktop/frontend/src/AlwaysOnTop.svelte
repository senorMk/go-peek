<script lang="ts">
  import { onMount } from 'svelte';
  import { desktop, onEvent, type PrototypeStatus } from './bridge';
  let pinned = $state(false), ready = $state(false), saving = $state(false), error = $state('');
  function update(status: PrototypeStatus) { pinned = status.alwaysOnTop; ready = true; }
  onMount(() => {
    const stop = onEvent<PrototypeStatus>('prototype:status', update);
    if (window.go) void desktop().GetStatus().then(update).catch(e => { error = String(e); });
    return stop;
  });
  async function toggle(event: Event) {
    const enabled = (event.currentTarget as HTMLInputElement).checked;
    const previous = pinned;
    pinned = enabled; saving = true; error = '';
    try { await desktop().SetAlwaysOnTop(enabled); update(await desktop().GetStatus()); }
    catch (e) { pinned = previous; error = String(e); }
    finally { saving = false; }
  }
</script>

<label class="checkbox"><input type="checkbox" checked={pinned} disabled={!ready || saving} onchange={toggle}/> Always on top</label>
{#if error}<p class="error" role="alert">{error}</p>{/if}
