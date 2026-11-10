<script lang="ts">
  import { formatBytes, formatRelative } from '../../lib/format';
  import { daysLeft, thumbSrc } from '../../lib/image';
  import { copyLink, type Library } from '../../lib/library.svelte';
  import { longpress } from '../../lib/longpress';
  import type { Image } from '../../lib/types';
  import Check from '../Check.svelte';
  import Icon from '../Icon.svelte';
  import RowMenu from './RowMenu.svelte';

  let { lib }: { lib: Library } = $props();

  const trash = $derived(lib.mode === 'trash');
  const showFolder = $derived(trash || lib.filtered);
  const allOn = $derived(lib.items.length > 0 && lib.items.every((i) => lib.sel.has(i.id)));
  const someOn = $derived(lib.items.some((i) => lib.sel.has(i.id)));

  function toggleAll() {
    if (allOn) lib.selectNone();
    else lib.selectAll();
  }

  function copy(im: Image) {
    void copyLink(im);
  }

  function rowKey(e: KeyboardEvent, id: string) {
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

<!-- svelte-ignore a11y_role_supports_aria_props_implicit -->
<table
  class="table"
  class:selecting={lib.selecting}
  aria-label={trash ? '回收站' : '图片'}
  aria-multiselectable="true"
>
  <colgroup>
    <col class="col-check" />
    <col class="col-thumb" />
    <col />
    {#if showFolder}<col class="col-folder" style="width: 20%" />{/if}
    <col class="col-dims" style="width: 112px" />
    <col class="col-size" style="width: 84px" />
    <col class="col-date" style="width: 112px" />
    <col class="col-act" style="width: 84px" />
  </colgroup>
  <thead>
    <tr>
      <th class="col-check">
        <Check on={allOn ? true : someOn ? 'mixed' : false} onchange={toggleAll} label="全选" class="cell-check" />
      </th>
      <th class="col-thumb"><span class="sr-only">预览</span></th>
      <th>文件名</th>
      {#if showFolder}<th class="col-folder">文件夹</th>{/if}
      <th class="col-dims">尺寸</th>
      <th class="col-size r">大小</th>
      <th class="col-date">{trash ? '清除' : '上传时间'}</th>
      <th class="col-act"><span class="sr-only">操作</span></th>
    </tr>
  </thead>
  <tbody>
    {#each lib.items as im (im.id)}
      {@const on = lib.sel.has(im.id)}
      {@const when = trash ? (daysLeft(im, lib.retention) ?? '') : formatRelative(im.createdAt)}
      {@const where = lib.label(im.folderId)}
      <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
      <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
      <tr
        aria-selected={on}
        class:is-open={lib.activeId === im.id}
        tabindex="0"
        data-id={im.id}
        draggable={!trash}
        onclick={(e) => lib.clickItem(im.id, e)}
        ondblclick={() => lib.openViewer(im.id)}
        onkeydown={(e) => rowKey(e, im.id)}
        ondragstart={(e) => lib.dragImages(im, e)}
        use:longpress={() => {
          lib.selectMode = true;
          lib.toggle(im.id);
          lib.suppressNextClick();
        }}
      >
        <td class="col-check">
          <Check on={on} onchange={(e) => lib.toggle(im.id, e.shiftKey)} label={`选择 ${im.filename}`} class="cell-check" />
        </td>
        <td class="col-thumb">
          {#if im.broken}
            <span class="thumb" style="display: grid; place-items: center"><Icon name="image" /></span>
          {:else}
            <img class="thumb" src={thumbSrc(im)} alt="" loading="lazy" draggable="false" />
          {/if}
        </td>
        <td>
          <div class="name-cell">
            <span class="name ellipsis">{im.filename}</span>
          </div>
          <div class="mobile-meta num">
            <span>{formatBytes(im.size)}</span>
            {#if showFolder}
              <span class="ellipsis">{where}</span>
            {:else}
              <span>{im.width} × {im.height}</span>
            {/if}
            <span>{when}</span>
          </div>
        </td>
        {#if showFolder}<td class="col-folder muted">{where}</td>{/if}
        <td class="col-dims muted num">{im.width} × {im.height}</td>
        <td class="col-size r muted num">{formatBytes(im.size)}</td>
        <td class="col-date muted num">{when}</td>
        <td class="col-act">
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div class="row-actions" onclick={(e) => e.stopPropagation()} ondblclick={(e) => e.stopPropagation()}>
            {#if trash}
              <button
                type="button"
                class="btn btn-ghost btn-sm btn-icon"
                aria-label={`恢复 ${im.filename}`}
                title="恢复"
                onclick={() => lib.restore([im.id])}
              >
                <Icon name="restore" />
              </button>
            {:else}
              <button
                type="button"
                class="btn btn-ghost btn-sm btn-icon"
                aria-label={`复制 ${im.filename} 的链接`}
                title="复制链接"
                onclick={() => copy(im)}
              >
                <Icon name="copy" />
              </button>
            {/if}
            <RowMenu {lib} image={im} />
          </div>
        </td>
      </tr>
    {/each}
  </tbody>
</table>
