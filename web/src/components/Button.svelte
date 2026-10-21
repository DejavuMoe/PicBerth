<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import type { IconName } from '../lib/icons';
  import Icon from './Icon.svelte';

  interface Props extends Omit<HTMLButtonAttributes, 'class' | 'children'> {
    variant?: 'primary' | 'secondary' | 'text' | 'danger' | 'inverse' | 'print';
    size?: 'md' | 'sm';
    icon?: IconName;
    block?: boolean;
    /** Renders an anchor (client-side navigation is handled by the router). */
    href?: string;
    /** With href: download the target instead of navigating (file name). */
    download?: string;
    children: Snippet;
  }

  let { variant = 'secondary', size = 'md', icon, block = false, href, download, children, type = 'button', ...rest }: Props =
    $props();
</script>

{#if href}
  <a class="btn {variant} {size}" class:block {href} {download}>
    {#if icon}<Icon name={icon} />{/if}
    {@render children()}
  </a>
{:else}
  <button class="btn {variant} {size}" class:block {type} {...rest}>
    {#if icon}<Icon name={icon} />{/if}
    {@render children()}
  </button>
{/if}

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    height: var(--control-h);
    padding: 0 14px;
    border: 1px solid transparent;
    border-radius: var(--r-ctl);
    font-size: 13.5px;
    font-weight: 600;
    line-height: 1;
    letter-spacing: 0.005em;
    white-space: nowrap;
    text-decoration: none;
    cursor: pointer;
    transition:
      background-color var(--dur) var(--ease),
      border-color var(--dur) var(--ease),
      color var(--dur) var(--ease),
      transform 80ms var(--ease);
  }
  .btn:active:not(:disabled) {
    transform: translateY(1px);
  }
  .btn:disabled {
    cursor: default;
    opacity: 0.4;
  }
  .sm {
    height: calc(var(--control-h) - 6px);
    padding: 0 10px;
    font-size: var(--t-13);
  }
  .block {
    display: flex;
    width: 100%;
  }

  .primary {
    background: var(--ink);
    color: var(--bg);
  }
  .primary:hover:not(:disabled) {
    background: color-mix(in srgb, var(--ink) 84%, var(--bg));
  }

  .secondary {
    border-color: var(--line-strong);
    color: var(--ink);
  }
  .secondary:hover:not(:disabled) {
    border-color: var(--ink);
    background: var(--surface);
  }

  .text {
    color: var(--ink-2);
    padding-inline: var(--s2);
  }
  .text:hover:not(:disabled) {
    background: var(--surface-2);
    color: var(--ink);
  }

  .danger {
    border-color: color-mix(in srgb, var(--danger) 45%, transparent);
    color: var(--danger);
  }
  .danger:hover:not(:disabled) {
    background: var(--danger);
    border-color: var(--danger);
    color: var(--accent-ink);
  }

  /* On film-dark surfaces (selection bar, toasts, lightbox). */
  .inverse {
    color: var(--print);
    padding-inline: 10px;
  }
  .inverse:hover:not(:disabled) {
    background: color-mix(in srgb, var(--print) 12%, transparent);
  }
  .inverse:focus-visible,
  .print:focus-visible {
    outline-color: var(--edge);
  }
  /* The primary action on film-dark surfaces. */
  .print {
    background: var(--print);
    color: var(--film);
  }
  .print:hover:not(:disabled) {
    background: #fff;
  }
</style>
