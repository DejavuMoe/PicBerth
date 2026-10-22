<script lang="ts">
  import { SORTS, type Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import MenuItem from '../MenuItem.svelte';

  let { lib }: { lib: Library } = $props();

  const current = $derived(SORTS.find((s) => s.key === lib.sort) ?? SORTS[0]!);
</script>

<Menu align="right" width={160} label="排序">
  {#snippet trigger(open, toggle)}
    <button
      type="button"
      class="btn btn-ghost btn-icon"
      aria-haspopup="menu"
      aria-expanded={open}
      aria-label={`排序：${current.label}`}
      title={`排序：${current.label}`}
      onclick={toggle}
    >
      <Icon name="sort" />
    </button>
  {/snippet}
  {#snippet children(close)}
    {#each SORTS as s (s.key)}
      <MenuItem selected={s.key === lib.sort} onclick={() => (lib.setSort(s.key), close())}>{s.label}</MenuItem>
    {/each}
  {/snippet}
</Menu>
