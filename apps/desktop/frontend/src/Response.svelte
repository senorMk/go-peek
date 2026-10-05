<script lang="ts">
  let { text }: { text: string } = $props();
  function inline(value: string) {
    const parts: { kind: 'text' | 'bold' | 'code'; text: string }[] = [];
    let offset = 0;
    for (const match of value.matchAll(/`([^`\n]+)`|\*\*([^\n]+?)\*\*/g)) {
      if (match.index > offset) parts.push({ kind: 'text', text: value.slice(offset, match.index) });
      parts.push({ kind: match[1] !== undefined ? 'code' : 'bold', text: match[1] ?? match[2] });
      offset = match.index + match[0].length;
    }
    if (offset < value.length) parts.push({ kind: 'text', text: value.slice(offset) });
    return parts;
  }
  function blocks(value: string) {
    const result: { code: boolean; language: string; text: string }[] = [];
    let code = false, language = '', lines: string[] = [];
    for (const line of value.split('\n')) {
      if ((!code && line.startsWith('```')) || (code && line.trim() === '```')) {
        if (lines.length) result.push({ code, language, text: lines.join('\n') });
        lines = []; code = !code; language = code ? line.slice(3).trim() : '';
      } else lines.push(line);
    }
    if (lines.length) result.push({ code, language, text: lines.join('\n') });
    return result;
  }
</script>
<div class="response">
  {#each blocks(text) as block}
    {#if block.code}
      {#if block.language}<span class="code-language">{block.language}</span>{/if}
      <pre><code>{block.text}</code></pre>
    {:else}<div class="response-text">{#each inline(block.text) as part}{#if part.kind === 'bold'}<strong>{part.text}</strong>{:else if part.kind === 'code'}<code class="inline-code">{part.text}</code>{:else}{part.text}{/if}{/each}</div>{/if}
  {/each}
</div>
