<script lang="ts">
  import { SORTS, type Library } from '../../lib/library.svelte';
  import { phone } from '../../lib/media.svelte';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import MenuItem from '../MenuItem.svelte';

  let { lib }: { lib: Library } = $props();
</script>

<Menu align="right" label="显示">
  {#snippet trigger(open, toggle)}
    <button
      type="button"
      class="btn btn-ghost btn-icon"
      aria-haspopup="menu"
      aria-expanded={open}
      aria-label={phone.matches ? '更多' : '显示'}
      title={phone.matches ? '更多' : '显示'}
      onclick={toggle}
    >
      <Icon name={phone.matches ? 'more' : lib.view === 'grid' ? 'grid' : 'list'} />
    </button>
  {/snippet}
  {#snippet children(close)}
    {#if phone.matches}
      <MenuItem
        icon="select"
        onclick={() => {
          lib.selectMode = true;
          close();
        }}>选择</MenuItem
      >
      <div class="menu-sep"></div>
    {/if}
    <div class="menu-group">视图</div>
    <MenuItem
      checked={lib.view === 'table'}
      onclick={() => {
        lib.setView('table');
        close();
      }}>列表</MenuItem
    >
    <MenuItem
      checked={lib.view === 'grid'}
      onclick={() => {
        lib.setView('grid');
        close();
      }}>网格</MenuItem
    >
    <div class="menu-sep"></div>
    <div class="menu-group">排序</div>
    {#each SORTS as s (s.key)}
      <MenuItem
        checked={lib.sort === s.key}
        onclick={() => {
          lib.setSort(s.key);
          close();
        }}>{s.label}</MenuItem
      >
    {/each}
  {/snippet}
</Menu>
