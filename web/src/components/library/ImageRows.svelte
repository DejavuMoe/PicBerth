<script lang="ts">
  import { formatBytes, formatRelative } from '../../lib/format';
  import { daysLeft } from '../../lib/image';
  import type { Library } from '../../lib/library.svelte';
  import { longpress } from '../../lib/longpress';
  import Thumb from '../Thumb.svelte';
  import CopyLink from './CopyLink.svelte';
  import RowMenu from './RowMenu.svelte';

  let { lib }: { lib: Library } = $props();

  const trash = $derived(lib.mode === 'trash');
</script>

<div class="rows" class:selecting={lib.selecting}>
  {#each lib.items as im (im.id)}
    {@const on = lib.sel.has(im.id)}
    {@const left = trash ? daysLeft(im, lib.retention) : null}
    <div
      class="row"
      class:selected={on}
      class:active={lib.activeId === im.id}
      use:longpress={() => {
        lib.selectMode = true;
        lib.toggle(im.id);
        lib.suppressNextClick();
      }}
    >
      <span class="row-check" aria-hidden={!lib.selecting}>
        <input
          type="checkbox"
          class="check"
          checked={on}
          tabindex={lib.selecting ? 0 : -1}
          onchange={() => lib.toggle(im.id)}
          aria-label={`选择 ${im.filename}`}
        />
      </span>
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
          <CopyLink image={im} />
        {/if}
      {/if}
    </div>
  {/each}
</div>
