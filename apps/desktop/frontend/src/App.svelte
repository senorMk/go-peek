<script lang="ts">
  import { onMount } from 'svelte';
  import languages from '../../internal/prompts/languages.json';
  import Capture from './Capture.svelte';
  import AlwaysOnTop from './AlwaysOnTop.svelte';
  import Screenshot from './Screenshot.svelte';
  import Response from './Response.svelte';
  import { desktop, onEvent, type StreamEvent, type Config, type Problem, type CapturedScreenshot, type CaptureEvent } from './bridge';
  let mode = $state<'work'|'capture'|'settings'>('work');
  let screenshots = $state<CapturedScreenshot[]>([]);
  let includeScreenshots = $state(true), imageSupported = $state(true);
  let config = $state<Config>({ version: 1, provider: 'openai', model: 'gpt-6-luna', alwaysOnTop: false, captureChecksEnabled: false });
  let draftProvider = $state('openai'), draftModel = $state('gpt-6-luna');
  let keyAvailable = $state(false), settingsMessage = $state(''), saving = $state(false);
  let apiKey = $state(''), keyMessage = $state('');
  $effect(() => { if (mode !== 'settings') { apiKey = ''; keyMessage = ''; } });
  let statement = $state(''), code = $state(''), examples = $state('');
  let language = $state('Go'), action = $state('hint');
  let testInput = $state(''), expected = $state(''), actual = $state('');
  let followup = $state(''), output = $state(''), error = $state('');
  let busy = $state(false), active = $state(''), sessionReady = $state(false);
  let completion = $state(''), usage = $state('');
  let extraContextOpen = $state(false);
  let captureBusy = $state(false), sessionVersion = $state(0);
  const tokenFormatter = new Intl.NumberFormat('en-US');
  let early: StreamEvent[] = [];
  const actions = [ ['hint','Hint'], ['explain','Explain approach'], ['review','Review code'], ['debug','Debug'], ['solution','Full solution'] ];
  async function preferences() {
    apiKey = ''; keyMessage = '';
    try {
      const prefs = await desktop().GetPreferences();
      config = prefs.config;
      if (mode === 'capture' && !config.captureChecksEnabled) mode = 'work'; draftProvider = config.provider; draftModel = config.model;
      keyAvailable = prefs.keyAvailable; imageSupported = prefs.imageSupported; settingsMessage = prefs.message ?? '';
    } catch (e) { settingsMessage = String(e); }
  }
  function apply(event: StreamEvent) {
    if (event.requestId !== active) return;
    if (event.kind === 'text') output += event.text ?? '';
    else {
      busy = false;
      if (event.kind === 'done') {
        sessionReady = true; completion = 'Complete';
        if (event.usage) usage = `${tokenFormatter.format(event.usage.inputTokens)} input / ${tokenFormatter.format(event.usage.outputTokens)} output tokens`;
      } else if (event.kind === 'error') { error = event.error ?? 'Provider request failed.'; completion = 'Interrupted'; }
      else completion = 'Cancelled';
    }
  }
  onMount(() => {
    const stop = onEvent<StreamEvent>('assistant:stream', event => {
      if (busy && !active) early.push(event); else apply(event);
    });
    const stopCapture = onEvent<CaptureEvent>('capture:result', event => {
      captureBusy = event.kind === 'started';
      if (event.kind === 'done' && mode === 'settings') mode = 'work';
    });
    if (window.go) void preferences();
    return () => { stop(); stopCapture(); };
  });
  async function send(isFollowup = false) {
    busy = true; active = ''; early = []; output = ''; error = ''; usage = ''; completion = 'Streaming…';
    if (!isFollowup) sessionReady = false;
    const problem: Problem = { screenshots: includeScreenshots ? screenshots.map(image => image.dataUrl) : [], statement, code, examples, language, action, testInput, expected, actual };
    try {
      active = isFollowup ? await desktop().AskFollowup(followup) : await desktop().StartResponse(problem);
      early.forEach(apply); early = [];
      if (isFollowup) followup = '';
    } catch (e) { busy = false; error = String(e); completion = 'Not sent'; }
  }
  async function cancel() { try { await desktop().Cancel(); } catch (e) { error = String(e); } }
  async function reset() {
    saving = true;
    try {
      await desktop().Reset();
      active = ''; early = []; busy = false; sessionReady = false; output = ''; error = ''; followup = ''; completion = ''; usage = '';
      statement = ''; code = ''; examples = ''; testInput = ''; expected = ''; actual = ''; action = 'hint';
      screenshots = []; includeScreenshots = true; extraContextOpen = false;
      sessionVersion++; mode = 'work';
    } catch (e) { error = String(e); }
    finally { saving = false; }
  }
  async function save() {
    saving = true; settingsMessage = '';
    try {
      await desktop().SavePreferences({ ...config, version: 1, provider: draftProvider, model: draftModel });
      active = ''; early = []; sessionReady = false; output = ''; followup = ''; completion = ''; usage = '';
      await preferences();
    } catch (e) { settingsMessage = String(e); }
    finally { saving = false; }
  }
  async function toggleCaptureChecks(event: Event) {
    const enabled = (event.currentTarget as HTMLInputElement).checked;
    const previous = config;
    config = { ...config, captureChecksEnabled: enabled }; saving = true; settingsMessage = '';
    try { config = await desktop().SetCaptureChecks(enabled); }
    catch (e) { config = previous; settingsMessage = String(e); }
    finally { saving = false; }
  }
  async function deleteKey() {
    apiKey = ''; keyMessage = ''; settingsMessage = '';
    saving = true;
    try { await desktop().DeleteKey(config.provider); sessionReady = false; await preferences(); }
    catch (e) { settingsMessage = String(e); }
    finally { saving = false; }
  }
  async function saveKey() {
    saving = true; settingsMessage = ''; keyMessage = '';
    try {
      await desktop().SaveKey(config.provider, apiKey);
      apiKey = '';
      await preferences();
      keyMessage = 'API key saved.';
    } catch (e) { settingsMessage = String(e); }
    finally { apiKey = ''; saving = false; }
  }
</script>

<div class="window-drag-region" aria-hidden="true"></div>
<nav aria-label="Workspace"><button class:secondary={mode !== 'work'} onclick={() => mode = 'work'}>Work</button>{#if config.captureChecksEnabled}<button class:secondary={mode !== 'capture'} onclick={() => mode = 'capture'} disabled={busy}>Capture checks</button>{/if}<button class:secondary={mode !== 'settings'} onclick={() => { mode = 'settings'; void preferences(); }} disabled={busy || saving}>Settings</button><button class="secondary" onclick={reset} disabled={saving || captureBusy || (busy && !active)}>Clear session</button></nav>
<div hidden={mode === 'settings'}><Screenshot bind:images={screenshots} disableDiscard={busy} resetVersion={sessionVersion}/></div>
{#if mode === 'settings'}
<main>
  <header><span class="eyebrow">GOPEEK</span><h1>Settings</h1><p>Preferences are saved on this Mac.</p></header>
  <section aria-label="Window settings">
    <h2>Window</h2>
    <AlwaysOnTop/>
    <p class="help">Keep GoPeek above other windows. Changes save immediately and are restored when you reopen the app.</p>
  </section>
  <section aria-label="Diagnostic settings">
    <h2>Diagnostics</h2>
    <label class="checkbox"><input type="checkbox" checked={config.captureChecksEnabled} disabled={saving || busy} onchange={toggleCaptureChecks}/> Show Capture checks</label>
    <p class="help">Display the capture-protection test workspace. Off by default; changes save immediately.</p>
  </section>
  <section class="provider-settings">
    <h2>Provider: {config.provider === 'zen' ? 'OpenCode Zen' : 'OpenAI'} · {config.model} · {keyAvailable ? 'key saved' : 'key needed'}</h2>
    <fieldset disabled={busy || saving}>
      <label for="provider">Provider</label>
      <select id="provider" bind:value={draftProvider} onchange={() => { draftModel = 'gpt-6-luna'; apiKey = ''; keyMessage = ''; }}><option value="openai">OpenAI</option><option value="zen">OpenCode Zen</option></select>
      <label for="model">Model</label>
      {#if draftProvider === 'zen'}
        <select id="model" bind:value={draftModel}><option>gpt-6-luna</option><option>gpt-6-sol</option><option>gpt-6.1-sol</option></select>
      {:else}<input id="model" bind:value={draftModel} placeholder="Responses-compatible model ID"/>{/if}
      <div class="actions"><button onclick={save}>Save and reset conversation</button><button class="secondary" onclick={preferences}>Refresh key status</button><button class="secondary" onclick={deleteKey} disabled={!keyAvailable}>Delete saved key</button></div>
      {#if draftProvider === config.provider}
        <label for="api-key">{keyAvailable ? 'Replace API key' : 'API key'}</label>
        <input id="api-key" type="password" bind:value={apiKey} autocomplete="off" autocapitalize="none" spellcheck="false" maxlength="4096" placeholder={keyAvailable ? 'Enter a new key to replace the saved key' : 'Paste your API key'}/>
        <div class="actions"><button onclick={saveKey} disabled={!apiKey.trim()}>Save API key</button></div>
      {:else}<p class="help">Save provider settings before adding its API key.</p>{/if}
    </fieldset>
    <p class="help">Keys are stored in macOS Keychain. The entry is cleared after saving; saved keys are never displayed.</p>
    {#if keyMessage}<p class="help" role="status">{keyMessage}</p>{/if}
  </section>
  {#if settingsMessage}<p class="error" role="alert">{settingsMessage}</p>{/if}
  {#if !keyAvailable}<p class="help">Add your provider key above to start. No API requests occur until you ask for a response.</p>{/if}

</main>
{:else if mode === 'capture' && config.captureChecksEnabled}<Capture/>{:else if screenshots.length > 0}
<main>
  <header><span class="eyebrow">GOPEEK</span><h1>Work through the problem.</h1><p>Ask for a nudge, review your code, or explore an approach.</p></header>
  <p class="help">{config.provider === 'zen' ? 'OpenCode Zen' : 'OpenAI'} · {config.model} · {keyAvailable ? 'key saved' : 'key needed'} · <button class="text-button" onclick={() => { mode = 'settings'; void preferences(); }} disabled={busy || saving}>Settings</button></p>
  {#if settingsMessage}<p class="error" role="alert">{settingsMessage}</p>{/if}
  {#if !keyAvailable}<p class="help">Open Settings to configure your provider and refresh key status.</p>{/if}
  <fieldset disabled={busy || saving}>
    <section>
      <div class="form-row"><div><label for="language">Language</label><select id="language" bind:value={language}>{#each languages as name}<option value={name}>{name}</option>{/each}</select></div><div><label for="action">Action</label><select id="action" bind:value={action}>{#each actions as [value,label]}<option {value}>{label}</option>{/each}</select></div></div>
      <details class="optional-context" bind:open={extraContextOpen}>
        <summary>Add details, examples, or code</summary>
        <div class="optional-fields">
      <label for="problem">Additional problem details (optional)</label><textarea id="problem" bind:value={statement} rows="4" placeholder="Add details or constraints missing from the screenshots…"></textarea>
      <label for="examples">Examples (optional)</label><textarea id="examples" bind:value={examples} rows="2"></textarea>
      <label for="code">Current code</label><textarea id="code" class="code-input" bind:value={code} rows="6" spellcheck="false" placeholder="Paste your implementation…"></textarea>
        </div>
      </details>
      {#if action === 'debug'}
        <label for="test-input">Failing input</label><textarea id="test-input" bind:value={testInput} rows="2"></textarea>
        <div class="form-row"><div><label for="expected">Expected output</label><textarea id="expected" bind:value={expected} rows="2"></textarea></div><div><label for="actual">Actual output</label><textarea id="actual" bind:value={actual} rows="2"></textarea></div></div>
      {/if}
    </section>
  </fieldset>
  <label class="checkbox screenshot-inclusion"><input type="checkbox" bind:checked={includeScreenshots} disabled={busy || saving}/> Include {screenshots.length} screenshot{screenshots.length === 1 ? '' : 's'} with this request</label>
  {#if includeScreenshots && !imageSupported}<p class="error" role="alert">This model is not enabled for image input. Select gpt-6-luna, gpt-6-sol, or gpt-6.1-sol in Provider, or uncheck screenshots and enter the problem as text.</p>{/if}
  <div class="actions"><button onclick={() => send()} disabled={busy || saving || !keyAvailable || (!statement.trim() && (!includeScreenshots || screenshots.length === 0))}>Ask for {actions.find(([value]) => value === action)?.[1].toLowerCase()}</button><button class="secondary" onclick={cancel} disabled={!busy || !active}>Cancel</button></div>
  <p class="help">Asking sends the included screenshots and entered context to {config.provider === 'zen' ? 'OpenCode Zen' : 'OpenAI'}. Text input: 24 KiB; text conversation: 64 KiB; screenshots: 32 MiB total. Oversized inputs are rejected without truncation.</p>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <section aria-label="Assistant response" aria-busy={busy}>
    <div class="section-heading"><h2>Response</h2><span>{completion}</span></div>
    {#if output}<Response text={output}/>{:else}<p class="empty">Your response will appear here.</p>{/if}
    {#if completion === 'Interrupted' || completion === 'Cancelled'}<p class="help">Partial output is not added to follow-up context; the last complete exchange is retained.</p>{/if}
    {#if usage}<p class="help">{usage}</p>{/if}
    <p class="help">Suggested tests are advice. No code or tests have been executed.</p>
  </section>
  <section>
    {#if sessionReady}<p class="help">Follow-ups use the screenshots from the last completed request. To use newly captured images, ask for a new response above. Clear session starts a new question and clears saved context.</p>{/if}
    <label for="followup">Follow-up on the current problem</label><textarea id="followup" bind:value={followup} rows="2" disabled={busy || !sessionReady} placeholder="Ask for a more specific hint or explore an edge case…"></textarea>
    <button class="secondary" onclick={() => send(true)} disabled={busy || saving || !sessionReady || !followup.trim()}>Ask follow-up</button>
  </section>
  <footer>⌘B shows/hides GoPeek. Sessions stay in memory; Clear session clears screenshots and context. Capture-tool checks remain pending.</footer>
</main>
{:else}
<main>
  <header><span class="eyebrow">GOPEEK</span><h1>Capture a problem to begin.</h1><p>Use Capture screen or Capture region above, or press ⌘⇧S. Your screenshot will appear for review before the problem workspace opens.</p></header>
</main>
{/if}
