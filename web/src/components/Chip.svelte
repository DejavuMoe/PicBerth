<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { IconName } from '../lib/icons';
  import Icon from './Icon.svelte';

  interface Props {
    tone?: 'neutral' | 'warn' | 'danger' | 'ok';
    icon?: IconName;
    title?: string;
    describedby?: string;
    /** When set the chip is a toggle button. */
    onclick?: () => void;
    pressed?: boolean;
    children: Snippet;
  }

  let { tone = 'neutral', icon, title, describedby, onclick, pressed, children }: Props = $props();
</script>

{#if onclick}
  <button type="button" class="chip {tone} toggle" class:pressed aria-pressed={pressed} {title} {onclick}>
    {#if icon}<Icon name={icon} size={14} />{/if}
    {@render children()}
  </button>
{:else}
  <span class="chip {tone}" {title} aria-describedby={describedby}>
    {#if icon}<Icon name={icon} size={14} />{/if}
    {@render children()}
  </span>
{/if}

<style>
  /* A stamp: a small mark with a hairline in its own colour. */
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    height: 20px;
    padding: 0 6px;
    border: 1px solid currentColor;
    border-radius: var(--r-chip);
    color: var(--ink-2);
    font-size: var(--t-12);
    font-weight: 550;
    line-height: 1;
    white-space: nowrap;
  }
  .warn {
    color: var(--warn);
  }
  .danger {
    color: var(--danger);
  }
  .ok {
    color: var(--ok);
  }
  .toggle {
    height: calc(var(--control-h) - 4px);
    padding: 0 10px;
    font-size: var(--t-13);
    border-color: var(--line-strong);
    color: var(--ink-2);
    cursor: pointer;
    border-radius: var(--r-ctl);
    transition:
      background-color var(--dur) var(--ease),
      border-color var(--dur) var(--ease),
      color var(--dur) var(--ease);
  }
  .toggle:hover {
    border-color: var(--ink);
    color: var(--ink);
  }
  .toggle.pressed {
    background: var(--ink);
    border-color: var(--ink);
    color: var(--bg);
  }
</style>
