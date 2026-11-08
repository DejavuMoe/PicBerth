<script lang="ts">
  import { copyLinks, type Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';

  interface Props {
    lib: Library;
    /** Phones: the trash action is an icon-only button; the row fills the floating bar. */
    compact?: boolean;
  }
  let { lib, compact = false }: Props = $props();

  const trash = $derived(lib.mode === 'trash');
  const ids = $derived([...lib.sel]);
  const selected = $derived(lib.items.filter((i) => lib.sel.has(i.id)));
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
