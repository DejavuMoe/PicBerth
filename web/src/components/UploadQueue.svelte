<script lang="ts">
  import { copyLinks } from '../lib/library.svelte';
  import { clearFinished, uploads, type UploadItem } from '../lib/upload.svelte';
  import Icon from './Icon.svelte';
  import UploadList from './UploadList.svelte';

  // The upload queue shown in place at the top of 最近 and the library (the floating dock is for other screens).
  let { open, folder = '' }: { open: (u: UploadItem) => void; folder?: string } = $props();

  const queue = $derived(uploads.items);
  const doneItems = $derived(queue.filter((u) => u.status === 'done' && u.image));
  const busy = $derived(queue.some((u) => u.status === 'queued' || u.status === 'uploading' || u.status === 'processing'));
  const title = $derived(busy ? '正在上传' : doneItems.length > 0 ? '上传完成' : '上传失败');
</script>

{#if queue.length > 0}
  <section class="upq" aria-label={title}>
    <div class="upq-head">
      <span class="medium">{title}</span>
      <span class="muted num">{doneItems.length} / {queue.length}</span>
      {#if folder}<span class="upq-dest muted"><Icon name="folder" size="s" /><span class="ellipsis">{folder}</span></span>{/if}
      <span class="grow"></span>
      {#if doneItems.length > 1}
        <button type="button" class="btn btn-ghost btn-sm" onclick={() => void copyLinks(doneItems.map((u) => u.image!))}>
          <Icon name="copy" size="s" />
          复制全部链接
        </button>
      {/if}
      {#if !busy}
        <button type="button" class="btn btn-ghost btn-sm btn-icon" aria-label="关闭" title="关闭" onclick={() => clearFinished()}>
          <Icon name="x" />
        </button>
      {/if}
    </div>
    <UploadList {open} />
  </section>
{/if}
