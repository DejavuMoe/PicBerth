<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { IconName } from '../lib/icons';
  import Icon from './Icon.svelte';

  interface Props {
    icon?: IconName;
    danger?: boolean;
    end?: string | number;
    /** Radio-style row with a leading tick column; omit for a plain menu item. */
    checked?: boolean;
    disabled?: boolean;
    onclick: () => void;
    children: Snippet;
  }
  let { icon, danger = false, end, checked, disabled = false, onclick, children }: Props = $props();
  const role = $derived(checked !== undefined ? 'menuitemradio' : 'menuitem');
</script>

<button
  type="button"
  {role}
  aria-checked={checked}
  class="menu-item"
  class:danger
  {disabled}
  {onclick}
>
  {#if checked !== undefined}
    <span class="tick" aria-hidden="true">{#if checked}<Icon name="check" />{/if}</span>
  {/if}
  {#if icon}
    <Icon name={icon} />
  {/if}
  <span class="ellipsis">{@render children()}</span>
  {#if end !== undefined}<span class="end">{end}</span>{/if}
</button>
