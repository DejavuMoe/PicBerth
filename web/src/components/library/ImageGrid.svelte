<script lang="ts">
  import { formatName } from '../../lib/image';
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
        <Thumb image={im} />
        <label class="tile-check">
          <input type="checkbox" class="check" checked={on} onclick={(e) => lib.toggle(im.id, e.shiftKey)} aria-label={`选择 ${im.filename}`} />
        </label>
        <span class="tile-badges">
          {#if im.frames > 1}<span class="chip-overlay">{formatName(im) === 'GIF' ? 'GIF' : '动图'}</span>{/if}
          {#if im.visibility === 'private'}
            <span class="chip-overlay" role="img" aria-label="私有" title="私有"><Icon name="lock" size={12} /></span>
          {/if}
        </span>
        {#if !trash}
          <span class="tile-copy"><CopyLink image={im} variant="overlay" /></span>
        {/if}
      </span>
      <span class="name truncate" title={im.filename}>{im.filename}</span>
    </div>
  {/each}
</div>
