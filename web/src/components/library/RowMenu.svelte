<script lang="ts">
  import { copyLink, downloadImage, type Library } from '../../lib/library.svelte';
  import type { Image } from '../../lib/types';
  import { LINK_FORMATS } from '../../lib/upload.svelte';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import MenuItem from '../MenuItem.svelte';
  import FolderTargets from './FolderTargets.svelte';

  let { lib, image }: { lib: Library; image: Image } = $props();

  const trash = $derived(lib.mode === 'trash');
  let view = $state<'main' | 'move'>('main');
</script>

<Menu align="right">
  {#snippet trigger(open, toggle)}
    <button
      type="button"
      class="btn btn-ghost btn-sm btn-icon more"
      aria-haspopup="menu"
      aria-expanded={open}
      aria-label={`${image.filename} 的更多操作`}
      title="更多"
      onclick={(e) => {
        e.stopPropagation();
        toggle();
      }}
    >
      <Icon name="more" />
    </button>
  {/snippet}
  {#snippet children(close)}
    {#if trash}
      <MenuItem
        danger
        icon="trash"
        onclick={() => {
          lib.requestPurge([image.id]);
          close();
        }}>永久删除</MenuItem
      >
    {:else if view === 'move'}
      <MenuItem icon="left" onclick={() => (view = 'main')}>移动到</MenuItem>
      <div class="menu-sep"></div>
      <FolderTargets
        currentId={image.folderId ?? null}
        onpick={(id) => void lib.moveImages([image.id], id)}
        {close}
      />
    {:else}
      {#each LINK_FORMATS as f (f.value)}
        <MenuItem
          onclick={() => {
            void copyLink(image, f.value);
            close();
          }}>复制 {f.label}</MenuItem
        >
      {/each}
      <div class="menu-sep"></div>
      <MenuItem icon="folder" onclick={() => (view = 'move')}>移动到…</MenuItem>
      <MenuItem
        icon="download"
        onclick={() => {
          downloadImage(image);
          close();
        }}>下载原图</MenuItem
      >
      <MenuItem
        icon="trash"
        onclick={() => {
          lib.trash([image.id]);
          close();
        }}>移到回收站</MenuItem
      >
    {/if}
  {/snippet}
</Menu>
