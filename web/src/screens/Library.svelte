<script lang="ts">
  import { onDestroy, onMount, untrack } from 'svelte';
  import Drawer from '../components/Drawer.svelte';
  import Icon from '../components/Icon.svelte';
  import DetailPanel from '../components/library/DetailPanel.svelte';
  import FilterSheet from '../components/library/FilterSheet.svelte';
  import ImageCollection from '../components/library/ImageCollection.svelte';
  import PurgeConfirm from '../components/library/PurgeConfirm.svelte';
  import Toolbar from '../components/library/Toolbar.svelte';
  import Sheet from '../components/Sheet.svelte';
  import { refreshFacets } from '../lib/facets.svelte';
  import { formatBytes } from '../lib/format';
  import { Library } from '../lib/library.svelte';
  import { phone, wide } from '../lib/media.svelte';
  import { isTypingTarget } from '../lib/modal';
  import { openPicker } from '../lib/picker';
  import { bottomBar } from '../lib/toast.svelte';
  import type { Image } from '../lib/types';
  import { onUploaded, uploadDialog } from '../lib/upload.svelte';

  let { mode }: { mode: 'library' | 'trash' } = $props();

  // The router remounts this screen when the mode changes, so the mode is fixed for one instance.
  const lib = new Library(untrack(() => mode));
  const trash = lib.mode === 'trash';

  /** How long the docked column takes to collapse; the panel keeps its last image until then. */
  const DOCK_MS = 260;

  let searchEl = $state<HTMLInputElement | null>(null);
  let filterSheet = $state(false);

  // The toolbar lays out by the width the page content gets, not by the viewport.
  let pageEl = $state<HTMLElement | null>(null);
  let pageWidth = $state(0);
  $effect(() => {
    const el = pageEl;
    if (!el) return;
    const measure = () => {
      const cs = getComputedStyle(el);
      pageWidth = Math.round(el.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight));
    };
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  });

  $effect(() => {
    lib.key;
    untrack(() => void lib.reload());
  });

  onMount(() => {
    void refreshFacets();
    if (trash) lib.loadRetention();
    return onUploaded((img) => {
      void refreshFacets();
      lib.addUploaded(img);
    });
  });
  onDestroy(() => {
    lib.destroy();
    bottomBar.active = false;
  });

  // On phones the fixed selection bar sits where toasts appear.
  $effect(() => {
    bottomBar.active = phone.matches && lib.selecting;
  });
  $effect(() => {
    if (!phone.matches) filterSheet = false;
  });

  // Wide screens dock the detail beside the list; narrower ones use a drawer, phones a full sheet.
  const canDock = $derived(wide.matches && !phone.matches);
  const docked = $derived(canDock && lib.active !== null);
  let dockImg = $state.raw<Image | null>(null);
  $effect(() => {
    const a = lib.active;
    if (a) {
      dockImg = a;
      return;
    }
    const t = setTimeout(() => (dockImg = null), DOCK_MS);
    return () => clearTimeout(t);
  });

  const bare = $derived(lib.phase === 'ready' && lib.items.length === 0 && !lib.filtered);
  const showFab = $derived(phone.matches && !trash && lib.phase === 'ready' && !bare && !lib.selecting);

  function onkeydown(e: KeyboardEvent) {
    if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey || isTypingTarget(e.target)) return;
    const overlays = document.querySelectorAll('[data-overlay]').length;
    const arrow = e.key === 'ArrowLeft' || e.key === 'ArrowRight';
    if (arrow) {
      // Arrows move through the images when the detail is docked, or is the only overlay open.
      if (lib.active && overlays === (canDock ? 0 : 1)) {
        e.preventDefault();
        if (e.key === 'ArrowLeft') {
          if (lib.prev) lib.open(lib.prev.id);
        } else void lib.step(1);
      }
      return;
    }
    if (overlays > 0) return;
    if (e.key === '/') {
      e.preventDefault();
      searchEl?.focus();
    } else if (e.key === 'Escape') {
      if (lib.selecting) lib.clearSel();
      else if (canDock && lib.active) lib.close();
    }
  }

  const head = $derived(lib.facets);
  const retention = $derived(lib.retention);
</script>

<svelte:window {onkeydown} />

<div class="workspace" class:docked>
  <div class="workspace-main">
    <div class="page lib-page" bind:this={pageEl}>
      <div class="page-head">
        <div>
          <h1>{trash ? '回收站' : '全部图片'}</h1>
          {#if head}
            <p class="num">
              {head.total} 张 · {formatBytes(head.bytes)}{#if trash && retention !== null}{` · 保留 ${retention} 天后自动永久删除`}{/if}
            </p>
          {/if}
        </div>
        {#if trash}
          {#if (head?.total ?? lib.items.length) > 0}
            <div class="page-actions">
              <button type="button" class="btn btn-danger" onclick={() => lib.emptyTrash()}>清空回收站</button>
            </div>
          {/if}
        {:else if !phone.matches}
          <div class="page-actions">
            <button type="button" class="btn btn-primary" title="上传 (U)" onclick={() => (uploadDialog.open = true)}>
              <Icon name="upload" />
              上传
            </button>
          </div>
        {/if}
      </div>

      <Toolbar {lib} width={pageWidth} bind:searchEl onopenfilters={() => (filterSheet = true)} />
      <ImageCollection {lib} />
    </div>
  </div>

  {#if canDock}
    <aside class="dock" aria-label="图片详情" inert={!lib.active}>
      <div class="dock-inner">
        {#if lib.active ?? dockImg}
          <DetailPanel {lib} image={(lib.active ?? dockImg)!} mode="dock" onclose={() => lib.close()} />
        {/if}
      </div>
    </aside>
  {/if}
</div>

{#if showFab}
  <button type="button" class="fab" aria-label="上传" onclick={openPicker}>
    <Icon name="upload" size={22} />
  </button>
{/if}

{#if lib.active && !canDock}
  {#if phone.matches}
    <Sheet label="图片详情" full onclose={() => lib.close()}>
      {#snippet children(close)}
        <DetailPanel {lib} image={lib.active!} mode="sheet" onclose={close} />
      {/snippet}
    </Sheet>
  {:else}
    <Drawer label="图片详情" width="420px" onclose={() => lib.close()}>
      {#snippet children(close)}
        <DetailPanel {lib} image={lib.active!} mode="drawer" onclose={close} />
      {/snippet}
    </Drawer>
  {/if}
{/if}

{#if filterSheet}
  <FilterSheet {lib} onclose={() => (filterSheet = false)} />
{/if}

{#if lib.purgeIds}
  <PurgeConfirm count={lib.purgeIds.length} oncancel={() => (lib.purgeIds = null)} onconfirm={() => lib.confirmPurge()} />
{/if}
