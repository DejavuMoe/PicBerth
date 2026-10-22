<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { IconName } from '../lib/icons';
  import Icon from './Icon.svelte';

  interface Props {
    icon?: IconName;
    danger?: boolean;
    end?: string | number;
    /** Multi-select row with a checkbox. */
    checked?: boolean;
    /** Single-select (radio) row with a tick. */
    selected?: boolean;
    onclick: () => void;
    children: Snippet;
  }
  let { icon, danger = false, end, checked, selected, onclick, children }: Props = $props();
  const role = $derived(checked !== undefined ? 'menuitemcheckbox' : selected !== undefined ? 'menuitemradio' : 'menuitem');
</script>

<button
  type="button"
  {role}
  aria-checked={checked ?? selected}
  class="menu-item"
  class:danger
  {onclick}
>
  {#if checked !== undefined}
    <span class="menu-check" class:on={checked} aria-hidden="true">{#if checked}<Icon name="check" size={12} />{/if}</span>
  {:else if selected !== undefined}
    <span class="menu-tick">{#if selected}<Icon name="check" />{/if}</span>
  {:else if icon}
    <Icon name={icon} />
  {:else}
    <span class="menu-tick"></span>
  {/if}
  <span class="truncate">{@render children()}</span>
  {#if end !== undefined}<span class="end">{end}</span>{/if}
</button>
