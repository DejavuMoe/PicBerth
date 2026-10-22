<script lang="ts">
  import { copyLinks, type Library } from '../../lib/library.svelte';
  import { phone } from '../../lib/media.svelte';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import MenuItem from '../MenuItem.svelte';

  let { lib }: { lib: Library } = $props();

  const ids = $derived([...lib.sel]);
  const trash = $derived(lib.mode === 'trash');
  const total = $derived(lib.items.length);
  const allOn = $derived(ids.length === total && total > 0);
  const m = $derived(phone.matches);
</script>

<div class="bulkbar" role="toolbar" aria-label="批量操作">
  <span class="count">{ids.length > 0 ? `已选 ${ids.length} 张` : '点按图片以选择'}</span>
  <button type="button" class="btn btn-ghost btn-sm" onclick={() => (allOn ? lib.clearSel() : lib.selectAll())}>
    {allOn ? '取消全选' : m ? '全选' : `全选 ${total} 张`}
  </button>
  <span class="grow"></span>
  {#if ids.length > 0}
    {#if trash}
      <button type="button" class="btn btn-secondary btn-sm" onclick={() => lib.restore(ids)}>
        <Icon name="restore" />
        恢复
      </button>
      <button type="button" class="btn btn-danger btn-sm" onclick={() => lib.requestPurge(ids)}>永久删除</button>
    {:else}
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        class:btn-icon={m}
        aria-label={m ? '复制链接' : undefined}
        title={m ? '复制链接' : undefined}
        onclick={() => copyLinks(lib.items.filter((i) => lib.sel.has(i.id)))}
      >
        <Icon name="copy" />
        {#if !m}复制链接{/if}
      </button>
      <Menu align="right" up={m}>
        {#snippet trigger(open, toggle)}
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            class:btn-icon={m}
            aria-label={m ? '可见性' : undefined}
            title={m ? '可见性' : undefined}
            aria-expanded={open}
            onclick={toggle}
          >
            <Icon name="lock" />
            {#if !m}可见性<Icon name="chevron-down" size={14} />{/if}
          </button>
        {/snippet}
        {#snippet children(close)}
          <MenuItem icon="globe" onclick={() => (lib.setVisibility(ids, 'public'), close())}>设为公开</MenuItem>
          <MenuItem icon="lock" onclick={() => (lib.setVisibility(ids, 'private'), close())}>设为私有</MenuItem>
        {/snippet}
      </Menu>
      <button
        type="button"
        class="btn btn-danger btn-sm"
        class:btn-icon={m}
        aria-label={m ? '移到回收站' : undefined}
        title={m ? '移到回收站' : undefined}
        onclick={() => lib.trash(ids)}
      >
        <Icon name="trash" />
        {#if !m}移到回收站{/if}
      </button>
    {/if}
  {/if}
  <button type="button" class="btn btn-ghost btn-icon btn-sm" aria-label="取消选择" title="取消选择 (Esc)" onclick={() => lib.clearSel()}>
    <Icon name="x" />
  </button>
</div>
