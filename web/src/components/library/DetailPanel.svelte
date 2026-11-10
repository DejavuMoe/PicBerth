<script lang="ts">
  import { untrack } from 'svelte';
  import { formatAbsolute, formatBytes } from '../../lib/format';
  import { daysLeft, formatName, sourceLabel, thumbSrc } from '../../lib/image';
  import { copyLink, copyValue, downloadImage, type Library } from '../../lib/library.svelte';
  import type { Image, LinkFormat } from '../../lib/types';
  import { withTransition } from '../../lib/transition';
  import { LINK_FORMATS } from '../../lib/upload.svelte';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import FolderTargets from './FolderTargets.svelte';

  interface Props {
    lib: Library;
    image: Image;
    /** Docked column beside the list, right drawer, or phone sheet. */
    variant: 'docked' | 'drawer' | 'sheet';
    /** In a drawer or sheet this is the overlay's animated close. */
    onclose: () => void;
  }
  let { lib, image, variant, onclose }: Props = $props();

  const trash = $derived(lib.mode === 'trash');
  const left = $derived(trash ? daysLeft(image, lib.retention) : null);
  const total = $derived(lib.items.length);

  let renaming = $state(false);
  let name = $state('');
  let nameEl = $state<HTMLInputElement | null>(null);
  let copied = $state<'' | LinkFormat>('');

  // A different image starts from a clean title, editor and copy mark.
  $effect(() => {
    image.id;
    untrack(() => {
      renaming = false;
      name = image.filename;
      copied = '';
    });
  });

  $effect(() => {
    if (!renaming || !nameEl) return;
    nameEl.focus();
    const dot = nameEl.value.lastIndexOf('.');
    nameEl.setSelectionRange(0, dot > 0 ? dot : nameEl.value.length);
  });

  let touch = { x: 0, y: 0 };

  function prev() {
    const p = lib.prev;
    if (p) lib.open(p.id);
  }

  function next() {
    void lib.step(1);
  }

  function beginRename() {
    if (trash) return;
    renaming = true;
  }

  async function saveName() {
    if (!renaming) return;
    const n = name.trim();
    renaming = false;
    if (n && n !== image.filename) await lib.rename(image.id, n);
    else name = image.filename;
  }

  function onRenameKey(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      void saveName();
    } else if (e.key === 'Escape') {
      e.stopPropagation();
      name = image.filename;
      renaming = false;
    }
  }

  function copyFormat(f: LinkFormat) {
    copied = f;
    void copyLink(image, f);
  }
</script>

<aside
  class="detail"
  class:drawer={variant === 'drawer'}
  class:sheet={variant === 'sheet'}
  aria-label={image.filename}
  ontouchstart={(e) => {
    const t = e.touches[0];
    if (t) touch = { x: t.clientX, y: t.clientY };
  }}
  ontouchend={(e) => {
    const t = e.changedTouches[0];
    if (!t) return;
    const dx = t.clientX - touch.x;
    const dy = t.clientY - touch.y;
    if (Math.abs(dx) > 60 && Math.abs(dx) > Math.abs(dy) * 1.5) {
      if (dx < 0) next();
      else prev();
    }
  }}
>
  <div class="detail-head">
    <button
      type="button"
      class="btn btn-ghost btn-icon"
      aria-label="上一张"
      title="上一张"
      disabled={lib.activeIndex <= 0}
      onclick={prev}
    >
      <Icon name="left" />
    </button>
    <button
      type="button"
      class="btn btn-ghost btn-icon"
      aria-label="下一张"
      title="下一张"
      disabled={!lib.canNext}
      onclick={next}
    >
      <Icon name="right" />
    </button>
    <span class="pos">{lib.activeIndex + 1} / {total}</span>
    <span class="grow"></span>
    <button type="button" class="btn btn-ghost btn-icon" aria-label="关闭" title="关闭" onclick={onclose}>
      <Icon name="x" />
    </button>
  </div>

  <div class="detail-body">
    <button
      type="button"
      class="preview"
      aria-label="查看大图"
      title="查看大图"
      onclick={() => withTransition(() => (lib.viewing = true))}
    >
      {#if image.broken}
        <Icon name="image" />
      {:else}
        <img src={thumbSrc(image)} alt="" style:view-transition-name={lib.viewing ? 'none' : 'pb-zoom'} />
      {/if}
    </button>

    <div class="title-edit">
      {#if renaming}
        <input
          bind:this={nameEl}
          class="input title-input"
          aria-label="文件名"
          autocomplete="off"
          spellcheck="false"
          value={name}
          oninput={(e) => (name = e.currentTarget.value)}
          onblur={() => void saveName()}
          onkeydown={onRenameKey}
        />
      {:else}
        <h2>
          <button
            type="button"
            class="title-btn"
            title="重命名"
            aria-label={`${image.filename}，重命名`}
            disabled={trash}
            onclick={beginRename}
          >
            {image.filename}
          </button>
        </h2>
      {/if}
      <div class="facts num">
        <span>{image.width} × {image.height}</span>
        <span>{formatBytes(image.size)}</span>
        <span>{formatName(image)}</span>
        {#if image.frames > 1}<span>动图 {image.frames} 帧</span>{/if}
        {#if left}<span class="text-warning">{left}</span>{/if}
      </div>
    </div>

    {#if !trash}
      <div class="links" role="group" aria-label="复制链接">
        {#each LINK_FORMATS as f (f.value)}
          <button
            type="button"
            class="link-row"
            class:copied={copied === f.value}
            aria-label={`复制 ${f.label}`}
            onclick={() => copyFormat(f.value)}
          >
            <span class="k">{f.label}</span>
            <span class="v mono ellipsis">{image.links[f.value]}</span>
            {#key copied === f.value}<Icon name={copied === f.value ? 'check' : 'copy'} size="s" />{/key}
          </button>
        {/each}
      </div>
    {/if}

    <dl class="props">
      <dt>文件夹</dt>
      {#if trash}
        <dd class="ellipsis">{lib.label(image.folderId)}</dd>
      {:else}
        <dd class="has-menu">
          <Menu align="left" label="文件夹">
            {#snippet trigger(open, toggle)}
              <button
                type="button"
                class="folder-btn"
                aria-label={`文件夹，${lib.label(image.folderId)}`}
                aria-haspopup="menu"
                aria-expanded={open}
                onclick={toggle}
              >
                <span class="ellipsis">{lib.label(image.folderId)}</span>
                <Icon name="down" size="s" />
              </button>
            {/snippet}
            {#snippet children(close)}
              <FolderTargets
                currentId={image.folderId ?? null}
                onpick={(id) => void lib.moveImages([image.id], id)}
                {close}
              />
            {/snippet}
          </Menu>
        </dd>
      {/if}
      <dt>来源</dt>
      <dd class="ellipsis">{sourceLabel(image.source)}</dd>
      <dt>上传时间</dt>
      <dd class="num">{formatAbsolute(image.createdAt)}</dd>
      {#if trash}
        <dt>删除时间</dt>
        <dd class="num">{image.trashedAt ? formatAbsolute(image.trashedAt) : '—'}</dd>
      {/if}
      <dt>ID</dt>
      <dd>
        <button type="button" class="copy-inline" aria-label="复制 ID" title="复制" onclick={() => void copyValue(image.id, 'ID')}>
          <span class="mono ellipsis">{image.id}</span>
        </button>
      </dd>
      <dt>SHA-256</dt>
      <dd>
        <button
          type="button"
          class="copy-inline"
          aria-label="复制 SHA-256"
          title="复制"
          onclick={() => void copyValue(image.sha256, 'SHA-256')}
        >
          <span class="mono ellipsis">{image.sha256}</span>
        </button>
      </dd>
    </dl>
  </div>

  <div class="detail-foot">
    {#if trash}
      <button type="button" class="btn btn-secondary" onclick={() => lib.restore([image.id])}>
        <Icon name="restore" />
        恢复
      </button>
      <button type="button" class="btn btn-ghost btn-danger-text" onclick={() => lib.requestPurge([image.id])}>
        永久删除
      </button>
    {:else}
      <button type="button" class="btn btn-ghost" onclick={() => downloadImage(image)}>
        <Icon name="download" />
        下载
      </button>
      <button type="button" class="btn btn-ghost btn-danger-text" onclick={() => lib.trash([image.id])}>
        <Icon name="trash" />
        移到回收站
      </button>
    {/if}
  </div>
</aside>
