<script lang="ts">
  import { SORTS, type Library, type ViewMode } from '../../lib/library.svelte';
  import { phone } from '../../lib/media.svelte';
  import type { IconName } from '../../lib/icons';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import MenuItem from '../MenuItem.svelte';

  let { lib }: { lib: Library } = $props();

  const VIEWS: { value: ViewMode; icon: IconName; label: string }[] = [
    { value: 'grid', icon: 'grid', label: '网格' },
    { value: 'list', icon: 'list', label: '列表' },
  ];
</script>

{#snippet sortItems(close: () => void)}
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

{#if phone.matches}
  <Menu align="right" label="更多">
    {#snippet trigger(open, toggle)}
      <button type="button" class="btn btn-ghost btn-icon" aria-haspopup="menu" aria-expanded={open} aria-label="更多" title="更多" onclick={toggle}>
        <Icon name="more" />
      </button>
    {/snippet}
    {#snippet children(close)}
      <MenuItem
        icon="select"
        onclick={() => {
          lib.selectMode = true;
          close();
        }}>选择</MenuItem
      >
      <div class="menu-sep"></div>
      <div class="menu-group">视图</div>
      {#each VIEWS as v (v.value)}
        <MenuItem
          checked={lib.view === v.value}
          onclick={() => {
            lib.setView(v.value);
            close();
          }}>{v.label}</MenuItem
        >
      {/each}
      <div class="menu-sep"></div>
      <div class="menu-group">排序</div>
      {@render sortItems(close)}
    {/snippet}
  </Menu>
{:else}
  <Menu align="right" label="排序">
    {#snippet trigger(open, toggle)}
      <button type="button" class="btn btn-ghost btn-icon" aria-haspopup="menu" aria-expanded={open} aria-label="排序" title="排序" onclick={toggle}>
        <Icon name="sort" />
      </button>
    {/snippet}
    {#snippet children(close)}
      {@render sortItems(close)}
    {/snippet}
  </Menu>
  <div class="seg seg-icons" role="group" aria-label="视图">
    {#each VIEWS as v (v.value)}
      <button type="button" aria-pressed={lib.view === v.value} aria-label={v.label} title={v.label} onclick={() => lib.setView(v.value)}>
        <Icon name={v.icon} size="s" />
      </button>
    {/each}
  </div>
{/if}
