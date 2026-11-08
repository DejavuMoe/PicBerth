<script lang="ts">
  import Icon from './Icon.svelte';

  interface Props {
    /** true = checked, false = unchecked, 'mixed' = indeterminate. */
    on: boolean | 'mixed';
    onchange: (e: MouseEvent) => void;
    label: string;
    /** Extra class on the hit area (e.g. to hide it until hover). */
    class?: string;
  }
  let { on, onchange, label, class: className = '' }: Props = $props();
</script>

<button
  type="button"
  class="check-hit {className}"
  role="checkbox"
  aria-checked={on === 'mixed' ? 'mixed' : on}
  aria-label={label}
  onclick={(e) => {
    e.stopPropagation();
    onchange(e);
  }}
>
  <span class="check" data-on={String(on)} aria-hidden="true">
    {#if on === 'mixed'}
      <Icon name="minus" />
    {:else if on}
      <Icon name="check" />
    {/if}
  </span>
</button>
