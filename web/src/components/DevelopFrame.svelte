<script lang="ts">
  // An upload of this session, shown as a print developing in the tray: the preview emerges as bytes
  // arrive, a finished one is archived, a failed one is fogged by a light leak (docs/ui.md §5.1).
  import { onDestroy } from 'svelte';
  import { copyText } from '../lib/clipboard';
  import { toast } from '../lib/toast.svelte';
  import type { LinkFormat } from '../lib/types';
  import { linkOf, removeItem, retry, type UploadItem } from '../lib/upload.svelte';
  import Icon from './Icon.svelte';
  import Middle from './Middle.svelte';

  interface Props {
    item: UploadItem;
    format: LinkFormat;
  }
  let { item, format }: Props = $props();

  // Development: the pale ghost while waiting, the bytes bring the picture up, the server fixes it.
  const dev = $derived(
    item.status === 'queued' ? 0 : item.status === 'uploading' ? 0.1 + item.progress * 0.75 : item.status === 'error' ? 0.55 : 1,
  );
  const pct = $derived(Math.round(item.progress * 100));
  const status = $derived.by(() => {
    switch (item.status) {
      case 'queued':
        return item.notBefore > Date.now() ? '排队 · 稍后重试' : '等待显影';
      case 'uploading':
        return `显影 ${pct}%`;
      case 'processing':
        return '定影…';
      case 'done':
        return item.image?.duplicate ? '已存在 · 原链接' : '已存档';
      default:
        return '曝光失败';
    }
  });

  let copied = $state(false);
  let timer: ReturnType<typeof setTimeout> | undefined;
  async function copy() {
    const v = linkOf(item, format);
    if (!v) return;
    const r = await copyText(v);
    if (r === 'copied') {
      copied = true;
      clearTimeout(timer);
      timer = setTimeout(() => (copied = false), 1500);
    } else {
      toast('当前连接不安全，无法写入剪贴板');
    }
  }
  onDestroy(() => clearTimeout(timer));

  // A different corner leaks for each upload.
  const leak = $derived(['0% 0%', '100% 0%', '100% 100%', '0% 100%'][item.id % 4]);
</script>

<div
  class="frame {item.status}"
  style:--dev={dev}
  style:--leak={leak}
  title={item.status === 'error' ? item.errorDetail || item.error || '' : item.image?.hasGps ? '照片包含拍摄位置（GPS），公开链接的访问者可以读取它' : undefined}
>
  <span class="edge top" aria-hidden="true">
    <span class="st">{status}</span>
    <span class="flags">
      {#if item.image?.hasGps}<Icon name="pin" size={11} />{/if}
      {#if item.visibility === 'private'}<Icon name="lock" size={11} />{/if}
    </span>
  </span>

  <span class="pic">
    {#if item.previewUrl}
      <img src={item.previewUrl} alt="" draggable="false" />
    {/if}
    {#if item.status === 'error'}<span class="fog"></span>{/if}
    {#if item.status === 'processing'}<span class="fix"></span>{/if}
  </span>

  {#if item.status === 'uploading'}
    <span class="meter" style:width="{pct}%" aria-hidden="true"></span>
  {/if}

  <span class="actions">
    {#if item.status === 'done'}
      <button type="button" class="act primary" onclick={copy} aria-label="复制 {item.name} 的链接">
        <Icon name={copied ? 'check' : 'copy'} size={14} />
        <span>{copied ? '已复制' : '复制链接'}</span>
      </button>
    {:else if item.status === 'error'}
      <span class="why">{item.error}</span>
      <span class="row">
        {#if item.retryable}
          <button type="button" class="act" onclick={() => retry(item.id)}>重试</button>
        {/if}
        <button type="button" class="act" onclick={() => removeItem(item.id)} aria-label="移除 {item.name}">移除</button>
      </span>
    {:else}
      <button type="button" class="act ghost" onclick={() => removeItem(item.id)} aria-label="取消上传 {item.name}" title="取消">
        <Icon name="close" size={13} />
      </button>
    {/if}
  </span>

  <span class="edge bottom" aria-hidden="true">
    <span class="name"><Middle text={item.name} tail={8} /></span>
  </span>
</div>

<style>
  .frame {
    position: relative;
    aspect-ratio: 3 / 2;
    min-width: 0;
    background: var(--film-2);
    transition: --dev 320ms var(--ease);
  }
  .pic {
    position: absolute;
    inset: 0;
    overflow: hidden;
  }
  img {
    position: absolute;
    inset: 0;
    margin: auto;
    width: auto;
    height: auto;
    max-width: 100%;
    max-height: 100%;
    user-select: none;
    /* A print in the developer: pale and flat at first, contrast and colour come up with --dev. */
    filter: grayscale(calc(1 - var(--dev))) brightness(calc(2.1 - 1.1 * var(--dev))) contrast(calc(0.22 + 0.78 * var(--dev)));
    opacity: calc(0.35 + 0.65 * var(--dev));
  }
  .done img {
    animation: fixed 700ms var(--ease) both;
  }
  @keyframes fixed {
    0% {
      filter: brightness(1.25) contrast(1.05);
    }
    100% {
      filter: none;
    }
  }
  .processing .fix {
    position: absolute;
    inset: 0;
    background: linear-gradient(100deg, transparent 30%, rgb(240 174 78 / 0.16) 50%, transparent 70%);
    background-size: 250% 100%;
    animation: fixer 1.3s linear infinite;
  }
  @keyframes fixer {
    from {
      background-position: 120% 0;
    }
    to {
      background-position: -120% 0;
    }
  }
  /* Light leak: an over-exposed orange fog spreading from one corner. */
  .error img {
    filter: grayscale(0.7) brightness(1.2) contrast(0.7);
    opacity: 0.55;
  }
  .fog {
    position: absolute;
    inset: 0;
    background:
      radial-gradient(circle at var(--leak), rgb(255 120 40 / 0.95), rgb(230 60 20 / 0.55) 32%, transparent 72%),
      linear-gradient(rgb(40 12 6 / 0.35), rgb(40 12 6 / 0.35));
    mix-blend-mode: screen;
    animation: leak 600ms var(--ease) both;
  }
  @keyframes leak {
    from {
      opacity: 0;
    }
  }

  .meter {
    position: absolute;
    left: 0;
    bottom: -3px;
    height: 2px;
    background: var(--edge);
    transition: width 200ms linear;
  }

  .edge {
    position: absolute;
    left: 1px;
    right: 1px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    height: 11px;
    font-family: var(--font-mono);
    font-stretch: 87.5%;
    font-size: var(--t-11);
    font-weight: 600;
    letter-spacing: 0.04em;
    line-height: 1;
    white-space: nowrap;
    pointer-events: none;
  }
  .top {
    top: -16px;
    color: var(--edge);
  }
  .bottom {
    bottom: -16px;
    color: var(--edge-2);
  }
  .flags {
    display: inline-flex;
    gap: 5px;
  }
  .done .top {
    color: #9fd8a8;
  }
  .error .top {
    color: #ff8a6a;
  }
  .processing .st {
    animation: blink 1s steps(2, jump-none) infinite;
  }
  @keyframes blink {
    50% {
      opacity: 0.45;
    }
  }
  .name {
    min-width: 0;
    flex: 1;
  }

  .actions {
    position: absolute;
    inset: auto 6px 6px 6px;
    z-index: 2;
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 6px;
  }
  .row {
    display: flex;
    gap: 4px;
  }
  .why {
    max-width: 100%;
    padding: 4px 7px;
    background: rgb(20 8 5 / 0.85);
    color: #ffd6c8;
    font-size: 12px;
    line-height: 1.35;
    border-radius: 2px;
  }
  .act {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 28px;
    padding: 0 10px;
    border-radius: 2px;
    background: rgb(20 17 14 / 0.86);
    color: var(--print);
    font-size: 12.5px;
    font-weight: 650;
    transition: background-color var(--dur) var(--ease);
  }
  .act:hover {
    background: var(--film);
  }
  .act.primary {
    background: var(--print);
    color: var(--film);
  }
  .act.primary:hover {
    background: #fff;
  }
  .act.ghost {
    width: 28px;
    padding: 0;
    justify-content: center;
    opacity: 0;
    transition: opacity var(--dur) var(--ease);
  }
  .frame:hover .act.ghost,
  .act.ghost:focus-visible {
    opacity: 1;
  }
  .act:focus-visible {
    outline: 2px solid var(--edge);
    outline-offset: 1px;
  }
</style>
