<script lang="ts">
  import { copyText } from '../../lib/clipboard';
  import { originalHref } from '../../lib/image';
  import { downloadImage, type Library } from '../../lib/library.svelte';
  import { toast } from '../../lib/toast.svelte';
  import type { Image } from '../../lib/types';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import MenuItem from '../MenuItem.svelte';

  let { lib, image, up = false }: { lib: Library; image: Image; up?: boolean } = $props();

  const trashed = $derived(image.state === 'trashed');

  async function copyId() {
    const r = await copyText(image.id);
    toast(r === 'copied' ? '已复制 ID' : '当前连接不安全，无法写入剪贴板');
  }
</script>

<Menu align="right" {up} width={180}>
  {#snippet trigger(open, toggle)}
    <button type="button" class="btn btn-ghost btn-icon btn-sm" aria-label={`${image.filename} 的更多操作`} aria-expanded={open} onclick={toggle}>
      <Icon name="more" />
    </button>
  {/snippet}
  {#snippet children(close)}
    {#if trashed}
      <MenuItem icon="restore" onclick={() => (close(), lib.restore([image.id]))}>恢复</MenuItem>
      <MenuItem icon="download" onclick={() => (close(), downloadImage(image))}>下载</MenuItem>
      <div class="menu-sep"></div>
      <MenuItem icon="trash" danger onclick={() => (close(), lib.requestPurge([image.id]))}>永久删除</MenuItem>
    {:else}
      <MenuItem icon="image" onclick={() => (close(), lib.open(image.id))}>查看详情</MenuItem>
      <MenuItem icon="external" onclick={() => (close(), window.open(originalHref(image), '_blank', 'noopener'))}>打开原图</MenuItem>
      <MenuItem icon="download" onclick={() => (close(), downloadImage(image))}>下载</MenuItem>
      <MenuItem icon="clipboard" onclick={() => (close(), copyId())}>复制 ID</MenuItem>
      <div class="menu-sep"></div>
      {#if image.visibility === 'public'}
        <MenuItem icon="lock" onclick={() => (close(), lib.setVisibility([image.id], 'private'))}>设为私有</MenuItem>
      {:else}
        <MenuItem icon="globe" onclick={() => (close(), lib.setVisibility([image.id], 'public'))}>设为公开</MenuItem>
      {/if}
      <MenuItem icon="trash" danger onclick={() => (close(), lib.trash([image.id]))}>移到回收站</MenuItem>
    {/if}
  {/snippet}
</Menu>
