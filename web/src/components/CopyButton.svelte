<script lang="ts">
  import { onDestroy } from 'svelte';
  import { copyText, type CopyResult } from '../lib/clipboard';
  import Button from './Button.svelte';
  import IconButton from './IconButton.svelte';

  interface Props {
    value: string;
    label?: string;
    /** Text control to select when the clipboard API is unavailable. */
    field?: HTMLInputElement | HTMLTextAreaElement | null;
    iconOnly?: boolean;
    size?: 'md' | 'sm';
    onresult?: (r: CopyResult) => void;
  }
  let { value, label = '复制', field = null, iconOnly = false, size = 'md', onresult }: Props = $props();

  let copied = $state(false);
  let timer: ReturnType<typeof setTimeout> | undefined;

  async function run() {
    const r = await copyText(value, field);
    onresult?.(r);
    if (r === 'copied') {
      copied = true;
      clearTimeout(timer);
      timer = setTimeout(() => (copied = false), 1500);
    }
  }
  onDestroy(() => clearTimeout(timer));
</script>

{#if iconOnly}
  <IconButton {size} icon={copied ? 'check' : 'copy'} label={copied ? '已复制' : label} onclick={run} />
{:else}
  <span class="wrap">
    <Button {size} icon={copied ? 'check' : undefined} onclick={run}>{copied ? '已复制' : label}</Button>
  </span>
{/if}
<span class="sr-only" role="status">{copied ? '已复制' : ''}</span>

<style>
  .wrap {
    display: inline-flex;
    min-width: 72px;
  }
  .wrap :global(.btn) {
    flex: 1;
  }
</style>
