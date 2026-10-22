<script lang="ts">
  import { formatAbsolute, formatBytes, formatRelative } from '../../lib/format';
  import { daysLeft } from '../../lib/image';
  import type { Library } from '../../lib/library.svelte';
  import LockMark from '../LockMark.svelte';
  import Thumb from '../Thumb.svelte';
  import CopyLink from './CopyLink.svelte';
  import RowMenu from './RowMenu.svelte';

  let { lib }: { lib: Library } = $props();

  const trash = $derived(lib.mode === 'trash');
  const items = $derived(lib.items);

  /** Clicks on a control inside the row belong to that control. */
  function rowClick(e: MouseEvent, id: string) {
    if (e.target instanceof Element && e.target.closest('button, a, input, label, .menu-wrap')) return;
    lib.clickItem(id, e);
  }
  function rowKey(e: KeyboardEvent, id: string) {
    if (e.target !== e.currentTarget) return;
    if (e.key === 'Enter') lib.open(id);
    if (e.key === ' ') {
      e.preventDefault();
      lib.toggle(id);
    }
  }
</script>

<div class="list-wrap">
  <div class="list" class:selecting={lib.selecting} role="grid" aria-multiselectable="true" aria-label={trash ? '回收站' : '全部图片'}>
    {#each items as im, idx (im.id)}
      {@const on = lib.sel.has(im.id)}
      <div
        class="lrow"
        class:trash
        class:selected={on}
        class:active={lib.activeId === im.id}
        role="row"
        tabindex="0"
        aria-selected={on}
        onclick={(e) => rowClick(e, im.id)}
        onkeydown={(e) => rowKey(e, im.id)}
      >
        <span role="gridcell" class="l-check">
          <input type="checkbox" class="check" checked={on} onclick={(e) => lib.toggle(im.id, e.shiftKey)} aria-label={`选择 ${im.filename}`} />
        </span>
        <span role="gridcell"><Thumb image={im} size="sm" /></span>
        <span role="gridcell" class="l-name">
          <span class="name truncate" title={im.filename}>{im.filename}</span>
          <LockMark visibility={im.visibility} />
          {#if im.tags.length > 0}
            <span class="tags">
              {#each im.tags as t (t)}<span class="tag">{t}</span>{/each}
            </span>
          {/if}
        </span>
        {#if !trash}
          <span role="gridcell" class="l-dim">{im.width}&#8201;×&#8201;{im.height}</span>
        {/if}
        <span role="gridcell" class="l-size">{formatBytes(im.size)}</span>
        {#if trash}
          <span role="gridcell" class="l-time" title={im.trashedAt ? formatAbsolute(im.trashedAt) : undefined}>
            {im.trashedAt ? formatRelative(im.trashedAt) : '—'}
          </span>
          <span role="gridcell" class="l-time l-last">{daysLeft(im, lib.retention) ?? ''}</span>
        {:else}
          <span role="gridcell" class="l-time l-last" title={formatAbsolute(im.createdAt)}>{formatRelative(im.createdAt)}</span>
        {/if}
        <span role="gridcell" class="l-actions">
          {#if trash}
            <button type="button" class="btn btn-ghost btn-sm" onclick={() => lib.restore([im.id])}>恢复</button>
          {:else}
            <CopyLink image={im} />
          {/if}
          <RowMenu {lib} image={im} up={items.length > 5 && idx >= items.length - 3} />
        </span>
      </div>
    {/each}
  </div>
</div>
