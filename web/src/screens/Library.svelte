<script lang="ts">
  import { onDestroy, onMount, tick, untrack } from 'svelte';
  import Drawer from '../components/Drawer.svelte';
  import AdjustSheet from '../components/library/AdjustSheet.svelte';
  import BulkBar from '../components/library/BulkBar.svelte';
  import DetailPanel from '../components/library/DetailPanel.svelte';
  import FilterChips from '../components/library/FilterChips.svelte';
  import ImageCollection from '../components/library/ImageCollection.svelte';
  import LibraryHead from '../components/library/LibraryHead.svelte';
  import PurgeConfirm from '../components/library/PurgeConfirm.svelte';
  import Sheet from '../components/Sheet.svelte';
  import { refreshFacets } from '../lib/facets.svelte';
  import { Library } from '../lib/library.svelte';
  import { phone, wide } from '../lib/media.svelte';
  import { isTypingTarget } from '../lib/modal';
  import { bottomBar } from '../lib/toast.svelte';
  import type { Image } from '../lib/types';
  import { onUploaded } from '../lib/upload.svelte';

  let { mode }: { mode: 'library' | 'trash' } = $props();

  // The router remounts this screen when the mode changes, so the mode is fixed for one instance.
  const lib = new Library(untrack(() => mode));
  const trash = lib.mode === 'trash';

  /** How long the docked column takes to collapse; the panel keeps its last image until then. */
  const DOCK_MS = 250;

  let searchEl = $state<HTMLInputElement | null>(null);
  let searchOpen = $state(false);
  let adjust = $state(false);

  // The header lays out by the width the list column gets, not by the viewport.
  let pageEl = $state<HTMLElement | null>(null);
  let pageWidth = $state(1200);
  $effect(() => {
    const el = pageEl;
    if (!el) return;
    const measure = () => (pageWidth = Math.round(el.clientWidth));
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
    if (!phone.matches) adjust = false;
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

  async function focusSearch() {
    searchOpen = true;
    await tick();
    searchEl?.focus();
  }

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
      void focusSearch();
    } else if (e.key === 'Escape') {
      if (lib.selecting) lib.clearSel();
      else if (canDock && lib.active) lib.close();
    }
  }
</script>

<svelte:window {onkeydown} />

<div class="lib" class:docked>
  <div class="page" bind:this={pageEl}>
    <LibraryHead {lib} width={pageWidth} bind:searchEl bind:searchOpen onadjust={() => (adjust = true)} />
    {#if !bare}<FilterChips {lib} />{/if}
    <div class="page-body">
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

{#if phone.matches && lib.selecting}
  <BulkBar {lib} compact floating />
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

{#if adjust}
  <AdjustSheet {lib} onclose={() => (adjust = false)} />
{/if}

{#if lib.purgeIds}
  <PurgeConfirm count={lib.purgeIds.length} oncancel={() => (lib.purgeIds = null)} onconfirm={() => lib.confirmPurge()} />
{/if}
