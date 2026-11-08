<script lang="ts">
  import { formatBytes, formatRelative } from '../../lib/format';
  import { daysLeft, thumbSrc } from '../../lib/image';
  import type { Library } from '../../lib/library.svelte';
  import { longpress } from '../../lib/longpress';
  import Check from '../Check.svelte';
  import Icon from '../Icon.svelte';

  let { lib }: { lib: Library } = $props();

  const trash = $derived(lib.mode === 'trash');

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

<div
  class="grid"
  class:selecting={lib.selecting}
  role="listbox"
  aria-multiselectable="true"
  aria-label={trash ? '回收站' : '图片'}
>
  {#each lib.items as im (im.id)}
    {@const on = lib.sel.has(im.id)}
    <div
      class="tile"
      class:is-open={lib.activeId === im.id}
      role="option"
      aria-selected={on}
      tabindex="0"
      data-id={im.id}
      onclick={(e) => lib.clickItem(im.id, e)}
      onkeydown={(e) => tileKey(e, im.id)}
      use:longpress={() => {
        lib.selectMode = true;
        lib.toggle(im.id);
        lib.suppressNextClick();
      }}
    >
      {#if im.broken}
        <div class="tile-img" style="display: grid; place-items: center"><Icon name="image" /></div>
      {:else}
        <img class="tile-img" src={thumbSrc(im)} alt="" loading="lazy" draggable="false" />
      {/if}
      <Check on={on} onchange={(e) => lib.toggle(im.id, e.shiftKey)} label={`选择 ${im.filename}`} />
      <div class="tile-meta">
        <span class="name ellipsis grow">{im.filename}</span>
      </div>
      <div class="tile-sub num">
        <span>{formatBytes(im.size)}</span>
        <span>{trash ? (daysLeft(im, lib.retention) ?? '') : formatRelative(im.createdAt)}</span>
      </div>
    </div>
  {/each}
</div>
