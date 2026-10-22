<script lang="ts">
  import type { IconName } from '../../lib/icons';
  import Icon from '../Icon.svelte';

  interface Props {
    label: string;
    /** Active filters behind this button; the count replaces the chevron. */
    n: number;
    open: boolean;
    toggle: () => void;
    icon?: IconName;
  }
  let { label, n, open, toggle, icon }: Props = $props();
</script>

<button type="button" class="btn btn-secondary filter-btn" class:active={n > 0} aria-expanded={open} onclick={toggle}>
  {#if icon}<Icon name={icon} />{/if}
  <span>{label}</span>
  {#if n > 0}<span class="filter-n">{n}</span>{:else}<Icon name="chevron-down" size={14} />{/if}
</button>
