<script lang="ts">
  import { rawUrl } from '../../lib/api';
  import { copyLink, type Library } from '../../lib/library.svelte';
  import { overlay } from '../../lib/modal';
  import { withTransition } from '../../lib/transition';
  import type { Image } from '../../lib/types';
  import Icon from '../Icon.svelte';

  interface Props {
    lib: Library;
    image: Image;
  }
  let { lib, image }: Props = $props();

  const trash = $derived(lib.mode === 'trash');
  let actual = $state(false);

  // Every image starts fitted to the window.
  $effect(() => {
    image.id;
    actual = false;
  });

  function close() {
    withTransition(() => (lib.viewing = false));
  }

  function prev() {
    const p = lib.prev;
    if (p) lib.open(p.id);
  }

  function next() {
    void lib.step(1);
  }
</script>

<div class="viewer" role="dialog" aria-modal="true" aria-label={image.filename} tabindex="-1" use:overlay={{ onescape: close }}>
  <div class="viewer-bar">
    <span class="name ellipsis">{image.filename}</span>
    <span class="meta">{image.width} × {image.height}</span>
    <span class="grow"></span>
    <button type="button" class="vbtn" aria-label="上一张" title="上一张" disabled={lib.activeIndex <= 0} onclick={prev}>
      <Icon name="left" />
    </button>
    <button type="button" class="vbtn" aria-label="下一张" title="下一张" disabled={!lib.canNext} onclick={next}>
      <Icon name="right" />
    </button>
    <span class="meta pos">{lib.activeIndex + 1} / {lib.items.length}</span>
    {#if !trash}
      <button type="button" class="vbtn" onclick={() => void copyLink(image)}>
        <Icon name="copy" />
        <span class="label">复制链接</span>
      </button>
    {/if}
    <button
      type="button"
      class="vbtn"
      aria-label={actual ? '适应窗口' : '原始大小'}
      title={actual ? '适应窗口' : '原始大小'}
      aria-pressed={actual}
      onclick={() => (actual = !actual)}
    >
      <Icon name="scan" />
    </button>
    <button type="button" class="vbtn" aria-label="关闭" title="关闭" onclick={close}>
      <Icon name="x" />
    </button>
  </div>
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="viewer-stage" class:actual onclick={(e) => e.target === e.currentTarget && close()}>
    {#if image.broken}
      <Icon name="image" />
    {:else}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
      <img
        src={rawUrl(image.id)}
        alt={image.filename}
        width={actual ? image.width : undefined}
        style:view-transition-name="pb-zoom"
        onclick={() => (actual = !actual)}
      />
    {/if}
  </div>
</div>
