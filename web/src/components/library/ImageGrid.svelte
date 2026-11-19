<script lang="ts">
  import { formatBytes } from '../../lib/format';
  import { daysLeft, thumbSrc } from '../../lib/image';
  import { copyLink, type Library } from '../../lib/library.svelte';
  import { longpress } from '../../lib/longpress';
  import { createGridLayout } from './grid-layout';
  import Check from '../Check.svelte';
  import Icon from '../Icon.svelte';

  let { lib }: { lib: Library } = $props();

  const trash = $derived(lib.mode === 'trash');
  const showFolder = $derived(trash || lib.filtered);

  // Width of the area the rows fill: a zero-height probe inside the padded grid.
  let width = $state(0);
  function measure(node: HTMLElement) {
    width = Math.floor(node.clientWidth);
    const ro = new ResizeObserver(([e]) => {
      if (e) width = Math.floor(e.contentRect.width);
    });
    ro.observe(node);
    return { destroy: () => ro.disconnect() };
  }

  const layout = createGridLayout();
  const rows = $derived(width ? layout(lib.items, width) : []);

  function tileKey(e: KeyboardEvent, id: string) {
    if (e.target !== e.currentTarget) return;
    if (e.key === 'Enter') {
      e.preventDefault();
      lib.open(id);
    } else if (e.key === ' ') {
      e.preventDefault();
      lib.toggle(id);
    }
  }
</script>

<div class="jgrid" class:selecting={lib.selecting} role="listbox" aria-multiselectable="true" aria-label={trash ? '回收站' : '图片'}>
  <div use:measure style="height: 0; margin: -8px 0"></div>
  {#each rows as row, i (i)}
    <div class="jrow" role="presentation">
      {#each row.items as { im, r } (im.id)}
        {@const on = lib.sel.has(im.id)}
        <div
          class="jt"
          class:is-open={lib.activeId === im.id}
          style:width={`${r * row.h}px`}
          role="option"
          aria-selected={on}
          aria-label={im.filename}
          tabindex="0"
          data-id={im.id}
          draggable={!trash}
          onclick={(e) => lib.clickItem(im.id, e)}
          ondblclick={() => lib.openViewer(im.id)}
          onkeydown={(e) => tileKey(e, im.id)}
          ondragstart={(e) => lib.dragImages(im, e)}
          use:longpress={() => {
            lib.selectMode = true;
            lib.toggle(im.id);
            lib.suppressNextClick();
          }}
        >
          <div class="jt-frame" style:height={`${row.h}px`}>
            {#if im.broken}
              <div class="jt-ph"><Icon name="image" /></div>
            {:else}
              <img src={thumbSrc(im)} alt="" loading="lazy" draggable="false" />
            {/if}
            {#if im.frames > 1}<span class="jt-badge">动图</span>{/if}
            <Check on={on} onchange={(e) => lib.toggle(im.id, e.shiftKey)} label={`选择 ${im.filename}`} />
            {#if !trash}
              <button
                type="button"
                class="jt-copy"
                aria-label={`复制 ${im.filename} 的链接`}
                title="复制链接"
                onclick={(e) => {
                  e.stopPropagation();
                  void copyLink(im);
                }}
                ondblclick={(e) => e.stopPropagation()}
              >
                <Icon name="copy" size="s" />
              </button>
            {/if}
          </div>
          <div class="jt-cap">
            <span class="name ellipsis">{im.filename}</span>
            <span class="meta num">
              {trash ? (daysLeft(im, lib.retention) ?? formatBytes(im.size)) : showFolder ? lib.label(im.folderId) : formatBytes(im.size)}
            </span>
          </div>
        </div>
      {/each}
    </div>
  {/each}
</div>
