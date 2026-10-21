<script lang="ts">
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import type { IconName } from '../lib/icons';
  import Icon from './Icon.svelte';

  interface Props extends Omit<HTMLButtonAttributes, 'class' | 'children' | 'title'> {
    icon: IconName;
    /** Accessible name and tooltip. */
    label: string;
    variant?: 'plain' | 'inverse';
    size?: 'md' | 'sm';
  }

  let { icon, label, variant = 'plain', size = 'md', type = 'button', ...rest }: Props = $props();
</script>

<button class="ib {variant} {size}" {type} aria-label={label} title={label} {...rest}>
  <Icon name={icon} />
</button>

<style>
  .ib {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: none;
    width: var(--control-h);
    height: var(--control-h);
    border-radius: var(--r-ctl);
    color: var(--ink-2);
    transition:
      background-color var(--dur) var(--ease),
      color var(--dur) var(--ease);
  }
  .sm {
    width: calc(var(--control-h) - 6px);
    height: calc(var(--control-h) - 6px);
  }
  .ib:hover:not(:disabled) {
    background: var(--surface-2);
    color: var(--ink);
  }
  .ib:disabled {
    opacity: 0.4;
  }
  .inverse {
    color: color-mix(in srgb, var(--print) 72%, transparent);
  }
  .inverse:hover:not(:disabled) {
    background: color-mix(in srgb, var(--print) 12%, transparent);
    color: var(--print);
  }
  .inverse:focus-visible {
    outline-color: var(--edge);
  }
</style>
