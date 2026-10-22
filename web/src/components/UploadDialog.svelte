<script lang="ts">
  import { formatBytes } from '../lib/format';
  import { copyLinks } from '../lib/library.svelte';
  import { openPicker } from '../lib/picker';
  import { toast } from '../lib/toast.svelte';
  import type { Visibility } from '../lib/types';
  import {
    clearFinished,
    currentVisibility,
    LINK_FORMATS,
    removeItem,
    retry,
    uploadDialog,
    uploads,
    type UploadItem,
  } from '../lib/upload.svelte';
  import CopyButton from './CopyButton.svelte';
  import Icon from './Icon.svelte';
  import Modal from './Modal.svelte';
  import Seg from './Seg.svelte';
  import Thumb from './Thumb.svelte';

  const items = $derived(uploads.items);
  const active = $derived(items.filter((u) => u.status === 'queued' || u.status === 'uploading' || u.status === 'processing').length);
  const done = $derived(items.filter((u) => u.status === 'done').length);
  const failed = $derived(items.filter((u) => u.status === 'error').length);
  const fmtLabel = $derived(LINK_FORMATS.find((f) => f.value === uploads.format)?.label ?? '');

  // Overall progress over the items that can still finish; failed ones do not drag it down.
  const pct = $derived.by(() => {
    const counted = items.filter((u) => u.status !== 'error');
    if (counted.length === 0) return 0;
    const sum = counted.reduce((n, u) => n + (u.status === 'uploading' ? u.progress * 100 : u.status === 'queued' ? 0 : 100), 0);
    return Math.round(sum / counted.length);
  });

  const isActive = (u: UploadItem) => u.status === 'queued' || u.status === 'uploading' || u.status === 'processing';

  function close() {
    uploadDialog.open = false;
  }
  function cancel(u: UploadItem) {
    removeItem(u.id);
    toast('已取消上传');
  }
  function copyAll() {
    void copyLinks(items.filter((u) => u.status === 'done' && u.image).map((u) => u.image!));
  }
</script>

<Modal title="上传图片" width="560px" fullOnPhone blocksDrop={false} onclose={close}>
  <button type="button" class="dropzone" onclick={openPicker}>
    <span class="dz-icon"><Icon name="upload" size={22} /></span>
    <span class="dz-main">拖放或粘贴图片，或 <span class="dz-link">选择文件</span></span>
  </button>

  {#if items.length > 0}
    <div class="up-summary">
      <div class="up-summary-row">
        <span>
          {active ? `正在上传 ${active} 个，已完成 ${done} 个` : `已完成 ${done} 个`}{failed ? `，失败 ${failed} 个` : ''}
        </span>
        {#if active > 0}<span class="num subtle">{pct}%</span>{/if}
      </div>
      <ul class="up-list">
        {#each items as u (u.id)}
          <li class="up-row up-{u.status}">
            {#if u.image}
              <Thumb image={u.image} />
            {:else}
              <Thumb src={u.previewUrl ?? ''} />
            {/if}
            <div class="up-info">
              <span class="name truncate" title={u.name}>{u.name}</span>
              {#if u.status === 'uploading'}
                {@const p = Math.round(u.progress * 100)}
                <div class="progress" role="progressbar" aria-valuenow={p} aria-valuemin={0} aria-valuemax={100} aria-label={`${u.name} 上传进度`}>
                  <span style:width="{p}%"></span>
                </div>
                <span class="sub num">{p}% · {formatBytes(Math.round(u.file.size * u.progress))} / {formatBytes(u.file.size)}</span>
              {:else if u.status === 'queued'}
                <span class="sub">等待上传</span>
              {:else if u.status === 'processing'}
                <div class="progress indeterminate"><span></span></div>
                <span class="sub">处理中</span>
              {:else if u.status === 'done'}
                {#if u.image?.duplicate}
                  <span class="sub">已存在相同图片，返回原链接</span>
                {:else}
                  <span class="sub ok">
                    <Icon name="check" size={12} />
                    已上传 · {formatBytes(u.file.size)}{u.image?.visibility === 'private' ? ' · 私有' : ''}
                  </span>
                {/if}
              {:else}
                <span class="sub err" title={u.errorDetail || undefined}>{u.error}</span>
              {/if}
            </div>
            <div class="up-act">
              {#if isActive(u)}
                <button type="button" class="btn btn-ghost btn-icon btn-sm" aria-label={`取消上传 ${u.name}`} title="取消上传" onclick={() => cancel(u)}>
                  <Icon name="x" />
                </button>
              {:else if u.status === 'done' && u.image}
                <CopyButton iconOnly small text={u.image.links[uploads.format]} label={`复制 ${fmtLabel}`} />
              {:else if u.status === 'error'}
                {#if u.retryable}
                  <button type="button" class="btn btn-secondary btn-sm" onclick={() => retry(u.id)}>
                    <Icon name="refresh" />
                    重试
                  </button>
                {:else}
                  <button type="button" class="btn btn-ghost btn-icon btn-sm" aria-label={`移除 ${u.name}`} title="移除" onclick={() => removeItem(u.id)}>
                    <Icon name="x" />
                  </button>
                {/if}
              {/if}
            </div>
          </li>
        {/each}
      </ul>
    </div>
  {/if}

  {#snippet footer(closeAnimated)}
    <Seg
      class="foot-start"
      label="上传后可见性"
      value={currentVisibility()}
      onchange={(v: Visibility) => (uploads.visibilityChoice = v)}
      options={[
        { key: 'public', label: '公开' },
        { key: 'private', label: '私有' },
      ]}
    />
    {#if done > 0}
      <button type="button" class="btn btn-ghost" onclick={clearFinished}>清除已完成</button>
    {/if}
    {#if done > 1}
      <button type="button" class="btn btn-secondary" onclick={copyAll}>
        <Icon name="copy" />
        复制全部链接
      </button>
    {/if}
    <button type="button" class="btn btn-primary" onclick={closeAnimated}>{active ? '在后台继续' : '完成'}</button>
  {/snippet}
</Modal>
