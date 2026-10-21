<script lang="ts">
  // One archived picture on a film strip. Edge print above and below carries its metadata.
  import { onDestroy } from 'svelte';
  import { rawUrl, thumbUrl } from '../lib/api';
  import { copyText } from '../lib/clipboard';
  import { formatBytes, formatDims } from '../lib/format';
  import { toast } from '../lib/toast.svelte';
  import type { Image, LinkFormat } from '../lib/types';
  import Grease from './Grease.svelte';
  import Icon from './Icon.svelte';
  import Middle from './Middle.svelte';

  interface Props {
    image: Image;
    /** Frame number printed on the edge (1-based within its roll). */
    number: number;
    selected: boolean;
    /** A selection exists: a plain click toggles instead of opening. */
    selecting: boolean;
    fresh?: boolean;
    /** Bottom-edge note that replaces the dimensions (e.g. trash countdown). */
    note?: string | null;
    muted?: boolean;
    linkFormat: LinkFormat;
    /** This frame's picture is the shared element of a lightbox view transition. */
    morph?: boolean;
    onactivate: (e: MouseEvent | KeyboardEvent) => void;
    ontoggle: (e: MouseEvent | KeyboardEvent | PointerEvent) => void;
  }
  let {
    image,
    number,
    selected,
    selecting,
    fresh = false,
    note = null,
    muted = false,
    linkFormat,
    morph = false,
    onactivate,
    ontoggle,
  }: Props = $props();

  // thumb → raw → nothing (per id, so a reused component never keeps a stale stage)
  let failures = $state({ id: '', n: 0 });
  const stage = $derived((failures.id === image.id ? failures.n : 0) + (image.hasThumb ? 0 : 1));
  const src = $derived(stage === 0 ? thumbUrl(image.id) : stage === 1 ? rawUrl(image.id) : '');
  let loadedSrc = $state('');

  const label = $derived(
    `${image.filename}，${formatDims(image.width, image.height)}，${formatBytes(image.size)}${image.visibility === 'private' ? '，私有' : ''}`,
  );

  let copied = $state(false);
  let copyTimer: ReturnType<typeof setTimeout> | undefined;
  async function copy() {
    const r = await copyText(image.links[linkFormat]);
    if (r === 'copied') {
      copied = true;
      clearTimeout(copyTimer);
      copyTimer = setTimeout(() => (copied = false), 1500);
    } else {
      toast('当前连接不安全，无法写入剪贴板');
    }
  }
  onDestroy(() => {
    clearTimeout(copyTimer);
    clearTimeout(pressTimer);
  });

  // Touch: a long press marks the frame, like circling it on the sheet.
  let pressTimer: ReturnType<typeof setTimeout> | undefined;
  let pressAt: { x: number; y: number } | null = null;
  let suppressClick = false;
  function pointerdown(e: PointerEvent) {
    if (e.pointerType !== 'touch') return;
    pressAt = { x: e.clientX, y: e.clientY };
    clearTimeout(pressTimer);
    pressTimer = setTimeout(() => {
      suppressClick = true;
      navigator.vibrate?.(8);
      ontoggle(e);
    }, 450);
  }
  function pointermove(e: PointerEvent) {
    if (pressAt && Math.hypot(e.clientX - pressAt.x, e.clientY - pressAt.y) > 8) cancelPress();
  }
  function cancelPress() {
    clearTimeout(pressTimer);
    pressAt = null;
  }
  function click(e: MouseEvent) {
    if (suppressClick) {
      suppressClick = false;
      e.preventDefault();
      return;
    }
    onactivate(e);
  }
  function keydown(e: KeyboardEvent) {
    if (e.ctrlKey || e.metaKey || e.altKey) return;
    if (e.key === ' ') {
      e.preventDefault(); // Space marks the frame instead of "clicking" it
      ontoggle(e);
    } else if (e.key === 'c' || e.key === 'C') {
      e.preventDefault();
      void copy();
    }
  }
</script>

<div class="frame" class:sel={selected} class:selecting class:muted class:broken={image.broken} data-frame={image.id}>
  <span class="edge top" aria-hidden="true">
    <span class="no">{number}<span class="tri">▸</span>{#if fresh}<span class="new">NEW</span>{/if}</span>
    <span class="flags">
      {#if image.broken}<span class="bad">!</span>{/if}
      {#if image.frames > 1}<span>ANIM</span>{/if}
      {#if image.hasGps}<span class="flag" title="含拍摄位置（GPS）"><Icon name="pin" size={11} /></span>{/if}
      {#if image.visibility === 'private'}<span class="flag" title="私有"><Icon name="lock" size={11} /></span>{/if}
    </span>
  </span>

  <button
    type="button"
    class="open"
    data-frame-btn
    aria-label={label}
    aria-pressed={selecting ? selected : undefined}
    onclick={click}
    onkeydown={keydown}
    onpointerdown={pointerdown}
    onpointermove={pointermove}
    onpointerup={cancelPress}
    onpointercancel={cancelPress}
    oncontextmenu={(e) => {
      if (pressAt || suppressClick) e.preventDefault();
    }}
  >
    {#if src && !image.broken}
      <img
        {src}
        alt=""
        loading="lazy"
        decoding="async"
        draggable="false"
        class:ready={loadedSrc === src}
        class:has-alpha={image.mime === 'image/png' || image.mime === 'image/webp' || image.mime === 'image/gif' || image.mime === 'image/avif'}
        style:view-transition-name={morph ? 'pb-print' : undefined}
        onload={() => (loadedSrc = src)}
        onerror={() => (failures = { id: image.id, n: (failures.id === image.id ? failures.n : 0) + 1 })}
      />
    {:else}
      <span class="void edge">{image.broken ? '文件缺失或损坏' : '无法预览'}</span>
    {/if}
  </button>

  <span class="tools">
    <button
      type="button"
      class="tool"
      title={copied ? '已复制' : '复制链接（C）'}
      aria-label={copied ? '已复制' : `复制 ${image.filename} 的链接`}
      onclick={copy}
    >
      <Icon name={copied ? 'check' : 'copy'} size={14} />
    </button>
    <button
      type="button"
      class="tool pick"
      class:on={selected}
      title={selected ? '取消圈选' : '圈选（空格）'}
      aria-label={selected ? `取消圈选 ${image.filename}` : `圈选 ${image.filename}`}
      aria-pressed={selected}
      onclick={(e) => ontoggle(e)}
    >
      <span class="ring"></span>
    </button>
  </span>

  {#if selected}<Grease shape="box" seed={image.id} />{/if}

  <span class="edge bottom" aria-hidden="true">
    <span class="name"><Middle text={image.filename} tail={8} /></span>
    <span class="dims">{note ?? formatDims(image.width, image.height)}</span>
  </span>
</div>

<style>
  .frame {
    position: relative;
    aspect-ratio: 3 / 2;
    min-width: 0;
    background: var(--film-2);
    container-type: inline-size;
  }
  .open {
    position: relative;
    display: block;
    width: 100%;
    height: 100%;
    overflow: hidden;
    cursor: zoom-in;
  }
  .selecting .open {
    cursor: pointer;
  }
  /* Contained in the frame like a contact print: never cropped, never enlarged past its pixels. */
  img {
    position: absolute;
    inset: 0;
    margin: auto;
    width: auto;
    height: auto;
    max-width: 100%;
    max-height: 100%;
    opacity: 0;
    transition:
      opacity 240ms var(--ease),
      filter 240ms var(--ease);
    user-select: none;
    -webkit-user-drag: none;
  }
  img.ready {
    opacity: 1;
  }
  /* Transparent pictures sit on a dark checkerboard, only under the picture itself. */
  img.has-alpha {
    background-color: #2a241e;
    background-image: conic-gradient(#332c25 25%, transparent 0 50%, #332c25 0 75%, transparent 0);
    background-size: 12px 12px;
  }
  .open:hover img.ready {
    filter: brightness(1.06);
  }
  .muted img.ready {
    filter: grayscale(0.85) brightness(0.8);
    opacity: 0.75;
  }
  .muted .open:hover img.ready,
  .muted .open:focus-visible img.ready {
    filter: none;
    opacity: 1;
  }
  .void {
    position: absolute;
    inset: 0;
    display: grid;
    place-items: center;
    color: var(--edge-2);
    opacity: 0.8;
  }
  .broken .void {
    color: var(--mark);
  }
  .open:focus-visible {
    outline: 2px solid var(--edge);
    outline-offset: 2px;
  }

  /* Edge print: lives in the band above / below the frame. */
  .edge {
    position: absolute;
    left: 1px;
    right: 1px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    height: 11px;
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
  @container (max-width: 190px) {
    .dims {
      display: none;
    }
  }
  .no {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    font-weight: 650;
  }
  .tri {
    font-size: 8px;
  }
  .new {
    margin-left: 6px;
    padding: 1px 3px;
    background: var(--edge);
    color: var(--film);
    font-size: 9px;
    font-weight: 750;
  }
  .flags {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .flag {
    display: inline-flex;
  }
  .bad {
    color: var(--mark);
    font-weight: 800;
  }
  .name {
    min-width: 0;
    flex: 1;
  }
  .dims {
    flex: none;
  }

  /* Hover tools inside the frame. */
  .tools {
    position: absolute;
    right: 6px;
    bottom: 6px;
    z-index: 2;
    display: flex;
    gap: 4px;
    opacity: 0;
    transform: translateY(3px);
    transition:
      opacity var(--dur) var(--ease),
      transform var(--dur) var(--ease);
  }
  .frame:hover .tools,
  .frame:focus-within .tools,
  .selecting .tools {
    opacity: 1;
    transform: none;
  }
  @media (hover: none) {
    .tools {
      display: none;
    }
  }
  .tool {
    display: grid;
    place-items: center;
    width: 28px;
    height: 28px;
    border-radius: 2px;
    background: rgb(20 17 14 / 0.82);
    color: var(--print);
    transition: background-color var(--dur) var(--ease);
  }
  .tool:hover {
    background: var(--film);
    color: #fff;
  }
  .tool:focus-visible {
    outline: 2px solid var(--edge);
    outline-offset: 1px;
  }
  .ring {
    width: 14px;
    height: 14px;
    border: 1.6px solid currentColor;
    border-radius: 50%;
  }
  .pick.on {
    background: var(--mark);
  }
  .pick.on .ring {
    border-color: #fff;
    background: radial-gradient(circle, #fff 0 3px, transparent 3.5px);
  }
  .selecting:not(.sel) .tools .tool:not(.pick) {
    display: none;
  }
</style>
