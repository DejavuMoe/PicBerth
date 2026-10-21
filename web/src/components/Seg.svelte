<script lang="ts" generics="T extends string">
  import type { IconName } from '../lib/icons';
  import Icon from './Icon.svelte';

  interface Option {
    key: T;
    label?: string;
    title?: string;
    icon?: IconName;
  }
  interface Props {
    options: Option[];
    value: T;
    onchange: (key: T) => void;
    label: string;
    icons?: boolean;
    fill?: boolean;
  }
  let { options, value, onchange, label, icons = false, fill = false }: Props = $props();
</script>

<div class="seg" class:seg-icons={icons} class:fill role="group" aria-label={label}>
  {#each options as o (o.key)}
    <button
      type="button"
      aria-pressed={value === o.key}
      title={o.title}
      aria-label={icons ? o.title : undefined}
      onclick={() => onchange(o.key)}
    >
      {#if icons && o.icon}<Icon name={o.icon} />{:else}{o.label}{/if}
    </button>
  {/each}
</div>
