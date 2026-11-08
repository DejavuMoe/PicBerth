<script lang="ts" generics="T extends string">
  import type { IconName } from '../lib/icons';
  import Icon from './Icon.svelte';

  interface Option {
    key: T;
    label?: string;
    /** Accessible name, required when the option shows only an icon. */
    title?: string;
    icon?: IconName;
  }
  interface Props {
    options: Option[];
    value: T;
    onchange: (key: T) => void;
    label: string;
    /** Full width of its container. */
    block?: boolean;
  }
  let { options, value, onchange, label, block = false }: Props = $props();
</script>

<div class="seg" class:seg-block={block} role="group" aria-label={label}>
  {#each options as o (o.key)}
    <button
      type="button"
      aria-pressed={value === o.key}
      aria-label={o.title ?? o.label}
      title={o.title ?? o.label}
      onclick={() => onchange(o.key)}
    >
      {#if o.icon}<Icon name={o.icon} size="s" />{/if}
      {#if o.label}{o.label}{/if}
    </button>
  {/each}
</div>
