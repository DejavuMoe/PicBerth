<script lang="ts">
  import { formatBytes, formatRelative } from '../../lib/format';
  import { daysLeft } from '../../lib/image';
  import type { Library } from '../../lib/library.svelte';
  import { longpress } from '../../lib/longpress';
  import { LINK_FORMATS, uploads } from '../../lib/upload.svelte';
  import CopyButton from '../CopyButton.svelte';
  import Thumb from '../Thumb.svelte';
  import RowMenu from './RowMenu.svelte';

  let { lib }: { lib: Library } = $props();

  const trash = $derived(lib.mode === 'trash');
  const fmtLabel = $derived(LINK_FORMATS.find((f) => f.value === uploads.format)?.label ?? '');
</script>

<div class="rows">
  {#each lib.items as im (im.id)}
    {@const on = lib.sel.has(im.id)}
    {@const left = trash ? daysLeft(im, lib.retention) : null}
    <div
      class="row"
      class:selected={on}
      use:longpress={() => {
        lib.selectMode = true;
        lib.toggle(im.id);
        lib.suppressNextClick();
      }}
    >
      {#if lib.selecting}
        <input type="checkbox" class="check" checked={on} onchange={() => lib.toggle(im.id)} aria-label={`选择 ${im.filename}`} />
      {/if}
      <button type="button" class="row-btn" onclick={(e) => lib.clickItem(im.id, e)}>
        <Thumb image={im} cover />
        <span class="row-main">
          <span class="name truncate">{im.filename}</span>
          <span class="sub truncate">
            {#if trash}
              {formatBytes(im.size)}{left ? ` · ${left}` : ''}
            {:else}
              {formatBytes(im.size)} · {formatRelative(im.createdAt)}{im.visibility === 'private' ? ' · 私有' : ''}{im.hasGps ? ' · 含位置' : ''}
            {/if}
          </span>
        </span>
      </button>
      {#if !lib.selecting}
        {#if trash}
          <RowMenu {lib} image={im} />
        {:else}
          <CopyButton iconOnly variant="ghost" text={im.links[uploads.format]} label={`复制 ${fmtLabel}`} />
        {/if}
      {/if}
    </div>
  {/each}
</div>
