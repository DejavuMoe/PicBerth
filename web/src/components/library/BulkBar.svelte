<script lang="ts">
  import { copyLinks, type Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import MenuItem from '../MenuItem.svelte';

  interface Props {
    lib: Library;
    /** Icon-only buttons, for narrow headers. */
    compact?: boolean;
    /** Phones: a bar above the tab bar instead of the header controls. */
    floating?: boolean;
  }
  let { lib, compact = false, floating = false }: Props = $props();

  const ids = $derived([...lib.sel]);
  const trash = $derived(lib.mode === 'trash');
  const total = $derived(lib.items.length);
  const allOn = $derived(ids.length === total && total > 0);
  const sm = $derived(floating ? 'btn-sm' : '');
</script>

<div class="bulk" class:bulk-floating={floating} role="toolbar" aria-label="批量操作">
  <span class="count">已选 {ids.length} 张</span>
  <button type="button" class="btn btn-ghost {sm}" onclick={() => (allOn ? lib.clearSel() : lib.selectAll())}>
    {allOn ? '取消全选' : '全选'}
  </button>
  {#if floating}<span class="grow"></span>{/if}
  {#if ids.length > 0}
    {#if trash}
      <button type="button" class="btn btn-ghost {sm}" onclick={() => lib.restore(ids)}>
        <Icon name="restore" />
        恢复
      </button>
      <button type="button" class="btn btn-danger-ghost {sm}" onclick={() => lib.requestPurge(ids)}>永久删除</button>
    {:else}
      <button
        type="button"
        class="btn btn-ghost {sm}"
        class:btn-icon={compact}
        aria-label={compact ? '复制链接' : undefined}
        title={compact ? '复制链接' : undefined}
        onclick={() => copyLinks(lib.items.filter((i) => lib.sel.has(i.id)))}
      >
        <Icon name="copy" />
        {#if !compact}复制链接{/if}
      </button>
      <Menu align="right" up={floating} label="可见性">
        {#snippet trigger(open, toggle)}
          <button
            type="button"
            class="btn btn-ghost {sm}"
            class:btn-icon={compact}
            aria-label={compact ? '可见性' : undefined}
            title={compact ? '可见性' : undefined}
            aria-haspopup="menu"
            aria-expanded={open}
            onclick={toggle}
          >
            <Icon name="lock" />
            {#if !compact}可见性<Icon name="chevron-down" size={14} />{/if}
          </button>
        {/snippet}
        {#snippet children(close)}
          <MenuItem icon="globe" onclick={() => (lib.setVisibility(ids, 'public'), close())}>设为公开</MenuItem>
          <MenuItem icon="lock" onclick={() => (lib.setVisibility(ids, 'private'), close())}>设为私有</MenuItem>
        {/snippet}
      </Menu>
      <button
        type="button"
        class="btn btn-danger-ghost {sm}"
        class:btn-icon={compact}
        aria-label={compact ? '移到回收站' : undefined}
        title={compact ? '移到回收站' : undefined}
        onclick={() => lib.trash(ids)}
      >
        <Icon name="trash" />
        {#if !compact}移到回收站{/if}
      </button>
    {/if}
  {/if}
  <button type="button" class="btn btn-ghost btn-icon {sm}" aria-label="取消选择" title="取消选择 (Esc)" onclick={() => lib.clearSel()}>
    <Icon name="x" />
  </button>
</div>
