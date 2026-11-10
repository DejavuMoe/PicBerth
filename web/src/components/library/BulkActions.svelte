<script lang="ts">
  import { copyLinks, type Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import FolderTargets from './FolderTargets.svelte';

  interface Props {
    lib: Library;
    /** Phones: the move and trash actions are icon-only; the row fills the floating bar. */
    compact?: boolean;
  }
  let { lib, compact = false }: Props = $props();

  const trash = $derived(lib.mode === 'trash');
  const ids = $derived([...lib.sel]);
  const selected = $derived(lib.items.filter((i) => lib.sel.has(i.id)));
  const commonFolder = $derived(
    selected.length > 0 && selected.every((i) => (i.folderId ?? null) === (selected[0]?.folderId ?? null))
      ? (selected[0]?.folderId ?? null)
      : undefined,
  );
</script>

{#if trash}
  <button type="button" class="btn btn-ghost" onclick={() => lib.restore(ids)}>
    <Icon name="restore" />
    恢复
  </button>
  {#if compact}<span class="grow"></span>{/if}
  <button type="button" class="btn btn-ghost btn-danger-text" onclick={() => lib.requestPurge(ids)}>
    永久删除
  </button>
{:else}
  <button type="button" class="btn btn-ghost" onclick={() => copyLinks(selected)}>
    <Icon name="copy" />
    复制链接
  </button>
  <Menu align="left">
    {#snippet trigger(open, toggle)}
      <button
        type="button"
        class="btn btn-ghost"
        class:btn-icon={compact}
        aria-label="移动到文件夹"
        title="移动"
        aria-haspopup="menu"
        aria-expanded={open}
        onclick={toggle}
      >
        <Icon name="folder" />
        {#if !compact}移动{/if}
      </button>
    {/snippet}
    {#snippet children(close)}
      <div class="menu-group">移动到</div>
      <FolderTargets currentId={commonFolder} onpick={(id) => void lib.moveImages(ids, id)} {close} />
    {/snippet}
  </Menu>
  {#if compact}<span class="grow"></span>{/if}
  <button
    type="button"
    class="btn btn-ghost"
    class:btn-icon={compact}
    aria-label={compact ? '移到回收站' : undefined}
    title={compact ? '移到回收站' : undefined}
    onclick={() => lib.trash(ids)}
  >
    <Icon name="trash" />
    {#if !compact}移到回收站{/if}
  </button>
{/if}
