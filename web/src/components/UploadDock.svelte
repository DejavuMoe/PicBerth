<script lang="ts">
  import { formatBytes } from '../lib/format';
  import { thumbSrc } from '../lib/image';
  import { copyLink, copyLinks } from '../lib/library.svelte';
  import { openImage } from '../lib/open.svelte';
  import { clearFinished, removeItem, retry, uploads, type UploadItem } from '../lib/upload.svelte';
  import Icon from './Icon.svelte';

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

  function copyOne(u: UploadItem) {
    if (u.image) void copyLink(u.image);
  }
</script>

{#if items.length > 0}
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
        {#each items as u (u.id)}
          {@const src = u.previewUrl ?? (u.image ? thumbSrc(u.image) : null)}
          <div class="up-row">
            {#if src}
              <img class="thumb" {src} alt="" />
            {:else}
              <span class="thumb" style="display: block"></span>
            {/if}
            <div class="up-name">
              <span class="ellipsis medium">{u.name}</span>
              {#if u.status === 'queued' || u.status === 'uploading'}
                <div
                  class="bar"
                  role="progressbar"
                  aria-label={u.name}
                  aria-valuenow={Math.round(u.progress * 100)}
                  aria-valuemin={0}
                  aria-valuemax={100}
                >
                  <i style:width={`${Math.round(u.progress * 100)}%`}></i>
                </div>
              {:else if u.status === 'processing'}
                <div class="bar indeterminate" role="progressbar" aria-label={u.name}>
                  <i></i>
                </div>
              {:else if u.status === 'error'}
                <span class="s text-danger" title={u.errorDetail || undefined}>{u.error}</span>
              {:else if u.status === 'done'}
                {#if u.image?.duplicate}
                  <span class="s muted">已存在，使用原有链接</span>
                {:else}
                  <span class="s muted facts">
                    <span>{formatBytes(u.file.size)}</span>
                    {#if u.folderId}<span class="ellipsis">{u.folderLabel}</span>{/if}
                  </span>
                {/if}
              {/if}
            </div>
            <div class="row" style:gap="2px">
              {#if u.status === 'queued'}
                <span class="s muted">等待中</span>
              {:else if u.status === 'uploading'}
                <span class="s muted num">{Math.round(u.progress * 100)}%</span>
              {:else if u.status === 'processing'}
                <span class="s muted">处理中</span>
              {/if}
              {#if u.status === 'queued' || u.status === 'uploading'}
                <button
                  type="button"
                  class="btn btn-ghost btn-sm btn-icon"
                  aria-label={`取消 ${u.name}`}
                  title="取消"
                  onclick={() => removeItem(u.id)}
                >
                  <Icon name="x" />
                </button>
              {/if}
              {#if u.status === 'error'}
                {#if u.retryable}
                  <button type="button" class="btn btn-ghost btn-sm" onclick={() => retry(u.id)}>重试</button>
                {/if}
                <button
                  type="button"
                  class="btn btn-ghost btn-sm btn-icon"
                  aria-label={`移除 ${u.name}`}
                  title="移除"
                  onclick={() => removeItem(u.id)}
                >
                  <Icon name="x" />
                </button>
              {/if}
              {#if u.status === 'done' && u.image}
                <button type="button" class="btn btn-ghost btn-sm" onclick={() => open(u)}>查看</button>
                <button type="button" class="btn btn-secondary btn-sm" onclick={() => copyOne(u)}>
                  <Icon name="copy" size="s" />
                  复制
                </button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </section>
{/if}
