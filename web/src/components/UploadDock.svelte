<script lang="ts">
  import { copyLinks } from '../lib/library.svelte';
  import { openImage } from '../lib/open.svelte';
  import { router } from '../lib/router.svelte';
  import { clearFinished, uploads, type UploadItem } from '../lib/upload.svelte';
  import Icon from './Icon.svelte';
  import UploadList from './UploadList.svelte';

  const items = $derived(uploads.items);
  const doneItems = $derived(items.filter((u) => u.status === 'done' && u.image));
  const busy = $derived(items.some((u) => u.status === 'queued' || u.status === 'uploading' || u.status === 'processing'));
  const title = $derived(busy ? '正在上传' : doneItems.length > 0 ? '上传完成' : '上传失败');

  let folded = $state(false);
  // New files open the panel again.
  let seen = 0;
  $effect(() => {
    const n = items.length;
    if (n > seen && items.some((u) => u.status === 'queued')) folded = false;
    seen = n;
  });

  function copyAll() {
    void copyLinks(doneItems.map((u) => u.image!));
  }

  function open(u: UploadItem) {
    if (u.image) void openImage(u.image.id, u.image.folderId ?? null);
  }

</script>

<!-- 最近 lists the queue in place, so the dock stays away there. -->
{#if items.length > 0 && router.path !== '/recent'}
  <section class="dock" aria-label="上传">
    <div class="dock-head">
      <span class="t">{title}</span>
      <span class="muted num">{doneItems.length} / {items.length}</span>
      <span class="grow"></span>
      {#if doneItems.length > 1}
        <button type="button" class="btn btn-ghost btn-sm" onclick={copyAll}>
          <Icon name="copy" size="s" />
          复制全部链接
        </button>
      {/if}
      <button
        type="button"
        class="btn btn-ghost btn-sm btn-icon"
        aria-label={folded ? '展开' : '收起'}
        title={folded ? '展开' : '收起'}
        aria-expanded={!folded}
        onclick={() => (folded = !folded)}
      >
        <Icon name={folded ? 'up' : 'down'} />
      </button>
      {#if !busy}
        <button type="button" class="btn btn-ghost btn-sm btn-icon" aria-label="关闭" title="关闭" onclick={() => clearFinished()}>
          <Icon name="x" />
        </button>
      {/if}
    </div>
    {#if !folded}
      <div class="dock-list">
        <UploadList {open} />
      </div>
    {/if}
  </section>
{/if}
