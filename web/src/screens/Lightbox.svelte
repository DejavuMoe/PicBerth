<script lang="ts">
  // The print under the lamp: a modal, always-dark view of one image with its sleeve of details.
  import Button from '../components/Button.svelte';
  import CopyButton from '../components/CopyButton.svelte';
  import Icon from '../components/Icon.svelte';
  import IconButton from '../components/IconButton.svelte';
  import Segmented from '../components/Segmented.svelte';
  import Switch from '../components/Switch.svelte';
  import { rawUrl, thumbUrl } from '../lib/api';
  import { formatAbsolute, formatBytes, formatDims, formatLabel, formatRelative } from '../lib/format';
  import { isTypingTarget, lockScroll } from '../lib/modal';
  import type { Image, LinkFormat, Visibility } from '../lib/types';
  import { LINK_FORMATS } from '../lib/upload.svelte';

  interface Props {
    image: Image | null;
    mode: 'library' | 'trash';
    /** "7 / 128" */
    position: string;
    canPrev: boolean;
    canNext: boolean;
    visibilityBusy: boolean;
    linkFormat: LinkFormat;
    /** Bottom-edge note for trashed images. */
    note?: string | null;
    onformat: (f: LinkFormat) => void;
    onprev: () => void;
    onnext: () => void;
    onclose: () => void;
    ontrash: () => void;
    onrestore: () => void;
    onpurge: () => void;
    onvisibility: (v: Visibility) => void;
  }
  let {
    image,
    mode,
    position,
    canPrev,
    canNext,
    visibilityBusy,
    linkFormat,
    note = null,
    onformat,
    onprev,
    onnext,
    onclose,
    ontrash,
    onrestore,
    onpurge,
    onvisibility,
  }: Props = $props();

  let el = $state<HTMLDialogElement | null>(null);
  const open = $derived(image !== null);

  $effect(() => {
    if (!el || !open) return;
    const dialog = el;
    if (!dialog.open) dialog.showModal();
    const unlock = lockScroll();
    return () => {
      unlock();
      if (dialog.open) dialog.close();
    };
  });

  function oncancel(e: Event) {
    e.preventDefault();
    onclose();
  }

  function onkeydown(e: KeyboardEvent) {
    if (!image || e.ctrlKey || e.metaKey || e.altKey || isTypingTarget(e.target)) return;
    // A confirm dialog above the lightbox owns the keyboard.
    if (document.querySelectorAll('dialog[open]').length > 1) return;
    if (e.key === 'ArrowLeft' && canPrev) {
      e.preventDefault();
      onprev();
    } else if (e.key === 'ArrowRight' && canNext) {
      e.preventDefault();
      onnext();
    }
  }

  // ---- the print ----------------------------------------------------------------------------
  const hasAlpha = $derived(!!image && /png|webp|gif|avif/.test(image.mime));
  let rawLoaded = $state('');
  const raw = $derived(image ? rawUrl(image.id) : '');

  // ---- loupe: hold the mouse button on the print ------------------------------------------------
  let printEl = $state<HTMLElement | null>(null);
  let loupe = $state<{ x: number; y: number; bx: number; by: number; bw: number; bh: number } | null>(null);
  const BORDER = 6;
  const LOUPE = 200;

  function loupeAt(e: PointerEvent) {
    if (!printEl || !image) return;
    const r = printEl.getBoundingClientRect();
    const dw = r.width - 2 * BORDER;
    const dh = r.height - 2 * BORDER;
    const x = Math.min(Math.max(e.clientX - r.left - BORDER, 0), dw);
    const y = Math.min(Math.max(e.clientY - r.top - BORDER, 0), dh);
    // 1:1 pixels when the print is shown smaller than that, otherwise a 2.5× glass.
    const m = Math.max(image.width / dw, 2.5);
    loupe = {
      x: e.clientX - r.left,
      y: e.clientY - r.top,
      bw: dw * m,
      bh: dh * m,
      bx: -(x * m - LOUPE / 2),
      by: -(y * m - LOUPE / 2),
    };
  }
  function printDown(e: PointerEvent) {
    if (e.pointerType !== 'mouse' || e.button !== 0 || !image || image.broken) return;
    e.preventDefault();
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    loupeAt(e);
  }
  function printMove(e: PointerEvent) {
    if (loupe) loupeAt(e);
  }
  function printUp() {
    loupe = null;
  }

  // ---- swipe on touch -----------------------------------------------------------------------
  let swipe: { x: number; y: number } | null = null;
  function stageDown(e: PointerEvent) {
    if (e.pointerType === 'touch') swipe = { x: e.clientX, y: e.clientY };
  }
  function stageUp(e: PointerEvent) {
    if (!swipe) return;
    const dx = e.clientX - swipe.x;
    const dy = e.clientY - swipe.y;
    swipe = null;
    if (Math.abs(dx) < 50 || Math.abs(dy) > 60) return;
    if (dx < 0 && canNext) onnext();
    else if (dx > 0 && canPrev) onprev();
  }

  function sourceLabel(img: Image): string {
    if (img.source.kind === 'token') return `令牌 · ${img.source.tokenName}`;
    if (img.source.kind === 'cli') return '命令行';
    return '网页';
  }

  let linkField = $state<HTMLInputElement | null>(null);
</script>

<svelte:document {onkeydown} />

<dialog bind:this={el} class="lightbox on-film" aria-label="图片详情" {oncancel} onclose={() => open && onclose()}>
  {#if image}
    <div class="layout">
      <section class="stage" onpointerdown={stageDown} onpointerup={stageUp} aria-label="图片">
        <header class="bar">
          <button type="button" class="back" onclick={onclose}>
            <Icon name="chevron-left" />
            <span>返回底片</span>
            <kbd>Esc</kbd>
          </button>
          <span class="pos edge">{position}</span>
          <span class="nav">
            <IconButton variant="inverse" icon="chevron-left" label="上一张 (←)" disabled={!canPrev} onclick={onprev} />
            <IconButton variant="inverse" icon="chevron-right" label="下一张 (→)" disabled={!canNext} onclick={onnext} />
          </span>
        </header>

        <div class="table">
          <div
            class="print"
            class:loupe-on={!!loupe}
            bind:this={printEl}
            style:--ar="{image.width} / {image.height}"
            style:--arn={image.width / image.height}
            style:--nw="{image.width}px"
            style:view-transition-name="pb-print"
            role="img"
            aria-label={image.filename}
            onpointerdown={printDown}
            onpointermove={printMove}
            onpointerup={printUp}
            onpointercancel={printUp}
          >
            <span class="paper" class:checker-dark={hasAlpha}>
              {#if image.broken}
                <span class="void edge">文件缺失或损坏</span>
              {:else}
                {#key image.id}
                  {#if image.hasThumb}
                    <img class="under" src={thumbUrl(image.id)} alt="" decoding="sync" draggable="false" />
                  {/if}
                  <img
                    class="full"
                    class:ready={rawLoaded === raw}
                    src={raw}
                    alt=""
                    draggable="false"
                    onload={() => (rawLoaded = raw)}
                  />
                {/key}
              {/if}
            </span>
            {#if loupe}
              <span
                class="glass"
                style:left="{loupe.x}px"
                style:top="{loupe.y}px"
                style:background-image="url({raw})"
                style:background-size="{loupe.bw}px {loupe.bh}px"
                style:background-position="{loupe.bx}px {loupe.by}px"
              ></span>
            {/if}
          </div>
        </div>
        <p class="hint edge">
          {#if !image.broken}<span class="hint-mouse"><Icon name="loupe" size={12} /> 按住查看细节</span>{/if}
          <span class="hint-touch">左右滑动切换</span>
        </p>
      </section>

      <aside class="sleeve" aria-label="详情">
        <div class="head">
          <h2>{image.filename}</h2>
          <p class="stamps">
            <span class="stamp" class:solid={image.visibility === 'public'}>{image.visibility === 'public' ? '公开' : '私有'}</span>
            {#if image.frames > 1}<span class="stamp">动图 · {image.frames} 帧</span>{/if}
            {#if image.hasGps}<span class="stamp warn"><Icon name="pin" size={11} /> 含位置</span>{/if}
            {#if image.broken}<span class="stamp danger">损坏</span>{/if}
            {#if note}<span class="stamp warn">{note}</span>{/if}
          </p>
        </div>

        {#if image.broken}
          <p class="note danger">服务器上找不到这张图片的文件，或文件已损坏。请到「状态」页运行完整检查。</p>
        {/if}

        {#if mode === 'library'}
          <section class="block">
            <h3>链接</h3>
            <Segmented label="链接格式" options={LINK_FORMATS} value={linkFormat} onchange={onformat} />
            <div class="linkrow">
              <input
                class="link mono"
                readonly
                value={image.links[linkFormat]}
                bind:this={linkField}
                aria-label="{linkFormat} 链接"
                onfocus={(e) => e.currentTarget.select()}
              />
              <CopyButton value={image.links[linkFormat]} field={linkField} label="复制" />
            </div>
            {#if image.visibility === 'private'}
              <p class="note warn">私有图片的链接在公开前无法访问</p>
            {/if}
          </section>

          <section class="block">
            <div class="switch-row">
              <span id="vis-label">
                公开访问
                <small>{image.visibility === 'public' ? '任何拿到链接的人都能看到' : '只有你登录后能看到'}</small>
              </span>
              <Switch
                checked={image.visibility === 'public'}
                labelledby="vis-label"
                disabled={visibilityBusy}
                onchange={(on) => onvisibility(on ? 'public' : 'private')}
              />
            </div>
            {#if image.hasGps}
              <p class="note warn"><Icon name="pin" size={13} /> 这张照片包含拍摄位置（GPS）。公开后，链接的访问者可以读取它。</p>
            {/if}
          </section>
        {/if}

        <section class="block">
          <h3>信息</h3>
          <dl class="info">
            <dt>尺寸</dt>
            <dd class="mono">{formatDims(image.width, image.height)}</dd>
            <dt>大小</dt>
            <dd class="mono" title="{image.size} B">{formatBytes(image.size)}</dd>
            <dt>格式</dt>
            <dd class="mono">{formatLabel(image.mime, image.ext)}</dd>
            <dt>上传</dt>
            <dd title={formatAbsolute(image.createdAt)}>{formatRelative(image.createdAt)}</dd>
            {#if image.trashedAt}
              <dt>删除</dt>
              <dd title={formatAbsolute(image.trashedAt)}>{formatRelative(image.trashedAt)}</dd>
            {/if}
            <dt>来源</dt>
            <dd>{sourceLabel(image)}</dd>
            <dt>ID</dt>
            <dd class="mono">{image.id}</dd>
            <dt>SHA-256</dt>
            <dd class="hash">
              <span class="mono hashval" title={image.sha256}>{image.sha256}</span>
              <CopyButton iconOnly size="sm" value={image.sha256} label="复制 SHA-256" />
            </dd>
          </dl>
        </section>

        <footer class="actions">
          {#if mode === 'library'}
            <Button variant="inverse" icon="download" href={raw} download={image.filename}>下载原图</Button>
            <Button variant="inverse" icon="trash" onclick={ontrash}>移到回收站</Button>
          {:else}
            <Button variant="print" icon="restore" onclick={onrestore}>恢复</Button>
            <Button variant="danger" onclick={onpurge}>永久删除</Button>
          {/if}
        </footer>
      </aside>
    </div>
  {/if}
</dialog>

<style>
  .lightbox {
    --lb-bg: #0c0a08;
    --lb-panel: #14110e;
    --lb-line: #2a241f;
    width: 100vw;
    height: 100dvh;
    margin: 0;
    padding: 0;
    border: 0;
    background: var(--lb-bg);
    color: var(--print);
    overflow: hidden;
  }
  .lightbox::backdrop {
    background: #0c0a08;
  }
  .lightbox[open] {
    animation: lb-in 220ms var(--ease) both;
  }
  @keyframes lb-in {
    from {
      opacity: 0;
    }
  }
  .layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 380px;
    height: 100%;
  }

  /* ---- stage ---- */
  .stage {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
  }
  .bar {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    height: 56px;
    padding: 0 12px;
  }
  .back {
    justify-self: start;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 34px;
    padding: 0 10px 0 6px;
    border-radius: var(--r-ctl);
    color: color-mix(in srgb, var(--print) 80%, transparent);
    font-size: var(--t-13);
    font-weight: 600;
  }
  .back:hover {
    background: rgb(255 255 255 / 0.07);
    color: var(--print);
  }
  .back kbd {
    margin-left: 4px;
    border-color: var(--lb-line);
    color: var(--edge-2);
  }
  .back:focus-visible {
    outline-color: var(--edge);
  }
  .pos {
    color: var(--edge);
    font-size: 12px;
  }
  .nav {
    justify-self: end;
    display: flex;
    gap: 4px;
  }
  .table {
    flex: 1;
    min-height: 0;
    display: grid;
    place-items: center;
    padding: 8px 40px 0;
    container-type: size;
  }
  .print {
    position: relative;
    width: min(100cqw, 100cqh * var(--arn), calc(var(--nw) + 12px));
    aspect-ratio: var(--ar);
    padding: 6px;
    background: var(--print);
    box-shadow:
      0 1px 0 rgb(255 255 255 / 0.06) inset,
      0 30px 80px -20px rgb(0 0 0 / 0.8);
    cursor: zoom-in;
    touch-action: pan-y;
    user-select: none;
  }
  .print.loupe-on {
    cursor: none;
  }
  .paper {
    position: relative;
    display: block;
    width: 100%;
    height: 100%;
    overflow: hidden;
  }
  .checker-dark {
    background-color: #e9e4da;
    background-image: conic-gradient(#d9d3c7 25%, transparent 0 50%, #d9d3c7 0 75%, transparent 0);
    background-size: 14px 14px;
  }
  .paper img {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: contain;
  }
  .full {
    opacity: 0;
    transition: opacity 260ms var(--ease);
  }
  .full.ready,
  .paper:not(:has(.under)) .full {
    opacity: 1;
  }
  .void {
    position: absolute;
    inset: 0;
    display: grid;
    place-items: center;
    background: var(--film-2);
    color: var(--mark);
  }
  .glass {
    position: absolute;
    z-index: 2;
    width: 200px;
    height: 200px;
    margin: -100px 0 0 -100px;
    border-radius: 50%;
    border: 3px solid var(--film);
    background-color: var(--print);
    background-repeat: no-repeat;
    box-shadow:
      0 0 0 1px rgb(255 255 255 / 0.25),
      0 18px 40px rgb(0 0 0 / 0.55);
    pointer-events: none;
  }
  .hint {
    display: flex;
    justify-content: center;
    gap: 6px;
    height: 44px;
    align-items: center;
    color: var(--edge-2);
    opacity: 0.8;
  }
  .hint-mouse {
    display: inline-flex;
    align-items: center;
    gap: 5px;
  }
  .hint-touch {
    display: none;
  }
  @media (pointer: coarse) {
    .hint-mouse {
      display: none;
    }
    .hint-touch {
      display: inline;
    }
  }

  /* ---- sleeve ---- */
  .sleeve {
    display: flex;
    flex-direction: column;
    gap: 22px;
    min-height: 0;
    overflow-y: auto;
    padding: 20px 22px 0;
    background: var(--lb-panel);
    border-left: 1px solid var(--lb-line);
    scrollbar-color: var(--lb-line) transparent;
  }
  .head h2 {
    font-size: 17px;
    font-weight: 650;
    line-height: 1.35;
    color: #fff;
    overflow-wrap: anywhere;
  }
  .stamps {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 10px;
  }
  .stamp {
    color: var(--edge-2);
  }
  .stamp.solid {
    background: var(--print);
    border-color: var(--print);
    color: var(--film);
  }
  .stamp.warn {
    color: var(--edge);
  }
  .stamp.danger {
    color: #ff7a5c;
  }
  .block {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  h3 {
    font-family: var(--font-mono);
    font-stretch: 87.5%;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.08em;
    color: var(--edge);
  }
  .linkrow {
    display: flex;
    gap: 6px;
  }
  .link {
    flex: 1;
    min-width: 0;
    height: var(--control-h);
    padding: 0 10px;
    border: 1px solid var(--lb-line);
    border-radius: var(--r-ctl);
    background: var(--lb-bg);
    color: var(--print);
    font-size: 12.5px;
  }
  .link:focus-visible {
    border-color: var(--edge);
    box-shadow: none;
  }
  .linkrow :global(.btn) {
    background: var(--print);
    color: var(--film);
    border-color: var(--print);
  }
  .switch-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    font-weight: 600;
  }
  .switch-row small {
    display: block;
    color: var(--edge-2);
    font-size: 12px;
    font-weight: 450;
  }
  .note {
    display: flex;
    align-items: flex-start;
    gap: 6px;
    font-size: 12.5px;
    line-height: 1.5;
  }
  .note :global(svg) {
    margin-top: 2px;
  }
  .note.warn {
    color: var(--edge);
  }
  .note.danger {
    color: #ff8a6a;
  }
  .info {
    display: grid;
    grid-template-columns: 64px minmax(0, 1fr);
    font-size: 13px;
  }
  .info dt,
  .info dd {
    padding: 7px 0;
    border-bottom: 1px solid var(--lb-line);
  }
  .info dt {
    color: var(--edge-2);
  }
  .info dd {
    min-width: 0;
    color: var(--print);
  }
  .hash {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .hashval {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .hash :global(.ib) {
    color: var(--edge-2);
  }
  .actions {
    position: sticky;
    bottom: 0;
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: auto;
    padding: 14px 0 18px;
    background: var(--lb-panel);
    border-top: 1px solid var(--lb-line);
  }

  @media (max-width: 760px) {
    .layout {
      grid-template-columns: 1fr;
      grid-template-rows: auto auto;
      height: auto;
      min-height: 100%;
    }
    .lightbox {
      overflow-y: auto;
    }
    .table {
      height: 58dvh;
      padding: 0 12px;
    }
    .sleeve {
      overflow: visible;
      border-left: 0;
      border-top: 1px solid var(--lb-line);
    }
    .back span {
      display: none;
    }
    .back kbd {
      display: none;
    }
  }
</style>
