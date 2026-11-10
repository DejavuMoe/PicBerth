<script lang="ts">
  import { onDestroy, onMount, tick, untrack } from 'svelte';
  import Drawer from '../components/Drawer.svelte';
  import Sheet from '../components/Sheet.svelte';
  import ActiveFilters from '../components/library/ActiveFilters.svelte';
  import BulkActions from '../components/library/BulkActions.svelte';
  import DetailPanel from '../components/library/DetailPanel.svelte';
  import FolderChips from '../components/library/FolderChips.svelte';
  import ImageList from '../components/library/ImageList.svelte';
  import LibraryHead from '../components/library/LibraryHead.svelte';
  import PurgeConfirm from '../components/library/PurgeConfirm.svelte';
  import Viewer from '../components/library/Viewer.svelte';
  import { refreshFacets } from '../lib/facets.svelte';
  import { Library } from '../lib/library.svelte';
  import { phone, wide } from '../lib/media.svelte';
  import { isTypingTarget } from '../lib/modal';
  import { openRequest } from '../lib/open.svelte';
  import { bottomBar } from '../lib/toast.svelte';
  import { tree } from '../lib/tree.svelte';
  import { onUploaded, setUploadTarget } from '../lib/upload.svelte';

  let { mode }: { mode: 'library' | 'trash' } = $props();

  // The router remounts this screen when the mode changes, so the mode is fixed for one instance.
  const lib = new Library(untrack(() => mode));
  const trash = lib.mode === 'trash';

  let searchEl = $state<HTMLInputElement | null>(null);
  let searching = $state(false);
  let scroller = $state<HTMLElement | null>(null);

  $effect(() => {
    lib.key;
    untrack(() => void lib.reload());
  });

  onMount(() => {
    void tree.ensure();
    void refreshFacets();
    // Images dropped on the sidebar tree move through this screen's list.
    tree.mover = (ids, to) => void lib.moveImages(ids, to);
    if (trash) lib.loadRetention();
    const off = onUploaded((img) => {
      void refreshFacets();
      void lib.refreshFolders();
      lib.addUploaded(img);
    });
    return () => {
      off();
      tree.mover = null;
    };
  });
  onDestroy(() => {
    lib.destroy();
    setUploadTarget(null);
    bottomBar.active = false;
  });

  // The upload target follows the folder on screen; other screens upload to the root.
  $effect(() => {
    if (lib.mode === 'library' && lib.foldersPhase === 'ready' && !lib.missing) {
      setUploadTarget({
        id: lib.folderId,
        label: lib.folderId ? lib.label(lib.folderId) : '图片',
      });
    } else {
      setUploadTarget(null);
    }
  });

  // 查看 on an upload and results of the palette open an image's detail, possibly after a folder change.
  $effect(() => {
    const id = openRequest.id;
    if (!id || lib.mode !== 'library') return;
    untrack(() => {
      openRequest.id = null;
      if (lib.phase === 'ready' && lib.items.some((i) => i.id === id)) lib.open(id);
      else lib.pendingOpen = id;
    });
  });

  // The phone selection bar sits where toasts appear.
  $effect(() => {
    bottomBar.active = phone.matches && lib.selecting;
  });
  $effect(() => {
    if (!phone.matches) searching = false;
  });

  // Wide screens dock the detail beside the list; narrower ones use a drawer, phones a sheet.
  const variant = $derived(wide.matches && !phone.matches ? 'docked' : phone.matches ? 'sheet' : 'drawer');
  const openImg = $derived(lib.active);

  async function focusSearch() {
    if (phone.matches) {
      searching = true;
      await tick();
    }
    searchEl?.focus();
  }

  function maybeLoadMore() {
    if (!scroller || !lib.cursor || lib.more === 'loading' || lib.phase !== 'ready') return;
    if (scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight < 600) void lib.loadMore();
  }

  $effect(() => {
    lib.items.length;
    lib.cursor;
    lib.phase;
    void tick().then(() => maybeLoadMore());
  });

  function onkeydown(e: KeyboardEvent) {
    if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey) return;
    const typing = isTypingTarget(e.target) || document.querySelector('.scrim') !== null;
    if (e.key === 'Escape') {
      if (typing) return;
      if (lib.active) lib.close();
      else if (lib.selecting) lib.clearSel();
      return;
    }
    if (typing) return;
    if (e.key === '/') {
      e.preventDefault();
      void focusSearch();
      return;
    }
    if (!lib.active || (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight')) return;
    // Arrows move through the images when the detail is docked, or is the only overlay open.
    const overlays = document.querySelectorAll('[data-overlay]').length;
    if (overlays !== (variant === 'docked' ? 0 : 1) + (lib.viewing ? 1 : 0)) return;
    e.preventDefault();
    if (e.key === 'ArrowLeft') {
      const p = lib.prev;
      if (p) lib.open(p.id);
    } else {
      void lib.step(1);
    }
  }
</script>

<svelte:window {onkeydown} />

{#if lib.missing}
  <div class="panel-main">
    <div class="head">
      <div class="head-title">
        <h1>图片</h1>
      </div>
    </div>
    <div class="state">
      <span class="t">文件夹不存在</span>
      <button type="button" class="btn btn-secondary" onclick={() => lib.openFolder(null)}>返回图片</button>
    </div>
  </div>
{:else}
  <div class="panel-main">
    <LibraryHead {lib} bind:searchEl bind:searching />
    <ActiveFilters {lib} omitQuery={phone.matches && searching} />
    <div class="scroll" bind:this={scroller} onscroll={maybeLoadMore}>
      {#if lib.view === 'grid' && lib.phase === 'ready'}
        <FolderChips {lib} />
      {/if}
      <ImageList {lib} />
      {#if lib.phase === 'ready'}
        {#if lib.cursor === null && lib.items.length > 12}
          <div class="loadmore">已显示全部</div>
        {:else if lib.more === 'error'}
          <div class="loadmore">
            <button type="button" class="btn btn-secondary btn-sm" onclick={() => lib.loadMore()}>加载失败，重试</button>
          </div>
        {/if}
      {/if}
    </div>
  </div>
{/if}

{#if openImg && variant === 'docked'}
  <DetailPanel {lib} image={openImg} variant="docked" onclose={() => lib.close()} />
{:else if openImg && variant === 'sheet'}
  <Sheet label={openImg.filename} onclose={() => lib.close()}>
    {#snippet children(close)}
      <DetailPanel {lib} image={openImg} variant="sheet" onclose={close} />
    {/snippet}
  </Sheet>
{:else if openImg}
  <Drawer label={openImg.filename} onclose={() => lib.close()}>
    {#snippet children(close)}
      <DetailPanel {lib} image={openImg} variant="drawer" onclose={close} />
    {/snippet}
  </Drawer>
{/if}

{#if openImg && lib.viewing}
  <Viewer {lib} image={openImg} />
{/if}

{#if phone.matches && lib.sel.size > 0}
  <div class="float-bar" role="toolbar" aria-label="批量操作">
    <BulkActions {lib} compact />
  </div>
{/if}

{#if lib.purgeIds}
  <PurgeConfirm {lib} />
{/if}
