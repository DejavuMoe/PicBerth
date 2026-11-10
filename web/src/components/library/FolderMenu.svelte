<script lang="ts">
  import { MAX_DEPTH, descendantIds, fitsUnder } from '../../lib/folders';
  import { tree } from '../../lib/tree.svelte';
  import type { Folder } from '../../lib/types';
  import MenuItem from '../MenuItem.svelte';
  import FolderTargets from './FolderTargets.svelte';

  interface Props {
    folder: Folder;
    nonEmpty: boolean;
    onnew: () => void;
    onrename: () => void;
    ondelete: () => void;
    close: () => void;
  }
  let { folder, nonEmpty, onnew, onrename, ondelete, close }: Props = $props();

  let view = $state<'main' | 'move'>('main');
  const full = $derived(tree.chain(folder.id).length >= MAX_DEPTH);
  const excluded = $derived(new Set([folder.id, ...descendantIds(tree.folders, folder.id)]));
</script>

{#if view === 'move'}
  <MenuItem icon="left" onclick={() => (view = 'main')}>移动到</MenuItem>
  <div class="menu-sep"></div>
  <FolderTargets
    currentId={folder.parentId ?? null}
    exclude={excluded}
    blocked={(id) => (fitsUnder(tree.folders, folder.id, id) ? '' : `最多 ${MAX_DEPTH} 层`)}
    onpick={(id) => tree.moveFolder(folder.id, id)}
    {close}
  />
{:else}
  <MenuItem
    icon="folder-plus"
    disabled={full}
    end={full ? `最多 ${MAX_DEPTH} 层` : undefined}
    onclick={() => {
      close();
      onnew();
    }}>新建子文件夹</MenuItem
  >
  <MenuItem
    icon="edit"
    onclick={() => {
      close();
      onrename();
    }}>重命名</MenuItem
  >
  <MenuItem icon="folder" onclick={() => (view = 'move')}>移动到…</MenuItem>
  <div class="menu-sep"></div>
  <MenuItem
    danger
    icon="trash"
    disabled={nonEmpty}
    end={nonEmpty ? '不为空' : undefined}
    onclick={() => {
      close();
      ondelete();
    }}>删除</MenuItem
  >
{/if}
