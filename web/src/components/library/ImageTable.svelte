<script lang="ts">
  import { formatAbsolute, formatBytes, formatRelative } from '../../lib/format';
  import { daysLeft, formatName, hasAlpha } from '../../lib/image';
  import type { Library } from '../../lib/library.svelte';
  import type { ImageSort } from '../../lib/types';
  import Icon from '../Icon.svelte';
  import Thumb from '../Thumb.svelte';
  import VisLabel from '../VisLabel.svelte';
  import CopyLink from './CopyLink.svelte';
  import RowMenu from './RowMenu.svelte';

  let { lib }: { lib: Library } = $props();

  const trash = $derived(lib.mode === 'trash');
  const items = $derived(lib.items);
  const allOn = $derived(items.length > 0 && items.every((i) => lib.sel.has(i.id)));
  const someOn = $derived(!allOn && items.some((i) => lib.sel.has(i.id)));

  let headBox = $state<HTMLInputElement | null>(null);
  $effect(() => {
    if (headBox) headBox.indeterminate = someOn;
  });

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

{#snippet sortHead(label: string, desc: ImageSort, asc: ImageSort, numeric: boolean, extra = '')}
  {@const on = lib.sort === desc || lib.sort === asc}
  <th class={`${numeric ? 'c-num' : ''} ${extra}`.trim() || undefined} aria-sort={on ? (asc === desc || lib.sort === asc ? 'ascending' : 'descending') : 'none'}>
    <button type="button" class:sorted={on} onclick={() => lib.setSort(lib.sort === desc ? asc : desc)}>
      {label}
      <span class="sort-ind"><Icon name={on && lib.sort === asc && asc !== desc ? 'chevron-up' : 'chevron-down'} size={12} /></span>
    </button>
  </th>
{/snippet}

<div class="table-wrap">
  <table class="table">
    <thead>
      <tr>
        <th class="c-check">
          <input
            bind:this={headBox}
            type="checkbox"
            class="check"
            checked={allOn}
            onchange={() => (allOn ? lib.clearSel() : lib.selectAll())}
            aria-label="全选"
          />
        </th>
        <th class="c-thumb"><span class="sr-only">预览</span></th>
        {@render sortHead('文件名', 'name', 'name', false)}
        <th class="c-num c-dim">尺寸</th>
        {@render sortHead('大小', 'large', 'small', true)}
        {#if trash}
          <th class="c-date">删除时间</th>
          <th>自动清除</th>
        {:else}
          <th class="c-vis">可见性</th>
          {@render sortHead('上传时间', 'new', 'old', false, 'c-date')}
        {/if}
        <th class="c-actions"><span class="sr-only">操作</span></th>
      </tr>
    </thead>
    <tbody>
      {#each items as im, idx (im.id)}
        {@const on = lib.sel.has(im.id)}
        {@const left = trash ? daysLeft(im, lib.retention) : null}
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions, a11y_no_noninteractive_tabindex -->
        <tr
          class:selected={on}
          class:active={lib.activeId === im.id}
          tabindex="0"
          onclick={(e) => rowClick(e, im.id)}
          onkeydown={(e) => rowKey(e, im.id)}
        >
          <td class="c-check">
            <input
              type="checkbox"
              class="check"
              checked={on}
              onclick={(e) => lib.toggle(im.id, e.shiftKey)}
              aria-label={`选择 ${im.filename}`}
            />
          </td>
          <td class="c-thumb"><Thumb image={im} size="sm" cover={!hasAlpha(im)} /></td>
          <td class="c-name">
            <span class="cell-name">
              <span class="name truncate" title={im.filename}>{im.filename}</span>
              <span class="sub">
                <span>{formatName(im)}</span>
                <span class="sub-dim num">{im.width} × {im.height}</span>
                {#if im.visibility === 'private'}
                  <span class="sub-vis"><Icon name="lock" size={12} />私有</span>
                {/if}
                {#if im.frames > 1}<span>动图</span>{/if}
                {#if im.hasGps}
                  <span class="gps" title="含位置信息"><Icon name="pin" size={12} />含位置</span>
                {/if}
                {#each im.tags as t (t)}<span class="tag">{t}</span>{/each}
              </span>
            </span>
          </td>
          <td class="c-num c-dim subtle">{im.width} × {im.height}</td>
          <td class="c-num">{formatBytes(im.size)}</td>
          {#if trash}
            <td class="nowrap subtle c-date" title={im.trashedAt ? formatAbsolute(im.trashedAt) : undefined}>
              {im.trashedAt ? formatRelative(im.trashedAt) : '—'}
            </td>
            <td class="nowrap">{#if left}<span class="badge">{left}</span>{/if}</td>
          {:else}
            <td class="c-vis"><VisLabel visibility={im.visibility} /></td>
            <td class="nowrap subtle c-date" title={formatAbsolute(im.createdAt)}>{formatRelative(im.createdAt)}</td>
          {/if}
          <td class="c-actions">
            <span class="row-actions">
              {#if trash}
                <button type="button" class="btn btn-secondary btn-sm" onclick={() => lib.restore([im.id])}>恢复</button>
              {:else}
                <CopyLink image={im} />
              {/if}
              <RowMenu {lib} image={im} up={items.length > 5 && idx >= items.length - 3} />
            </span>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
