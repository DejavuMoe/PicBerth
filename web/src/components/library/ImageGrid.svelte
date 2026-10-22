<script lang="ts">
  import { formatBytes, formatRelative } from '../../lib/format';
  import { daysLeft, formatName } from '../../lib/image';
  import type { Library } from '../../lib/library.svelte';
  import { longpress } from '../../lib/longpress';
  import Icon from '../Icon.svelte';
  import Thumb from '../Thumb.svelte';
  import CopyLink from './CopyLink.svelte';

  let { lib }: { lib: Library } = $props();

  const trash = $derived(lib.mode === 'trash');

  function tileKey(e: KeyboardEvent, id: string) {
    if (e.target !== e.currentTarget) return;
    if (e.key === 'Enter') lib.open(id);
    if (e.key === ' ') {
      e.preventDefault();
      lib.toggle(id);
    }
  }
  function tileClick(e: MouseEvent, id: string) {
    if (e.target instanceof Element && e.target.closest('button, label, input')) return;
    lib.clickItem(id, e);
  }
</script>

<div class="grid" class:selecting={lib.selecting}>
  {#each lib.items as im (im.id)}
    {@const on = lib.sel.has(im.id)}
    {@const left = trash ? daysLeft(im, lib.retention) : null}
    <div
      class="tile"
      class:selected={on}
      class:active={lib.activeId === im.id}
      role="button"
      tabindex="0"
      aria-label={im.filename}
      aria-pressed={on}
      onclick={(e) => tileClick(e, im.id)}
      onkeydown={(e) => tileKey(e, im.id)}
      use:longpress={() => {
        lib.selectMode = true;
        lib.toggle(im.id);
        lib.suppressNextClick();
      }}
    >
      <span class="tile-media">
        <Thumb image={im} cover />
        <label class="tile-check">
          <input type="checkbox" class="check" checked={on} onclick={(e) => lib.toggle(im.id, e.shiftKey)} aria-label={`选择 ${im.filename}`} />
        </label>
        <span class="tile-badges">
          {#if im.visibility === 'private'}
            <span class="badge badge-overlay"><Icon name="lock" size={12} />私有</span>
          {/if}
          {#if im.frames > 1}<span class="badge badge-overlay">{formatName(im) === 'GIF' ? 'GIF' : '动图'}</span>{/if}
          {#if im.hasGps}
            <span class="badge badge-overlay" title="含位置信息" aria-label="含位置信息"><Icon name="pin" size={12} /></span>
          {/if}
        </span>
        {#if !trash}
          <span class="tile-copy"><CopyLink image={im} variant="overlay" /></span>
        {/if}
      </span>
      <span class="tile-meta">
        <span class="name truncate" title={im.filename}>{im.filename}</span>
        <span class="sub truncate">
          {#if trash}
            {formatBytes(im.size)}{left ? ` · ${left}` : ''}
          {:else}
            {formatName(im)} · {formatBytes(im.size)} · {formatRelative(im.createdAt)}
          {/if}
        </span>
      </span>
    </div>
  {/each}
</div>
