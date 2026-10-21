<script lang="ts">
  import { onDestroy, onMount, untrack } from 'svelte';
  import Drawer from '../components/Drawer.svelte';
  import Icon from '../components/Icon.svelte';
  import DetailPanel from '../components/library/DetailPanel.svelte';
  import FilterSheet from '../components/library/FilterSheet.svelte';
  import ImageCollection from '../components/library/ImageCollection.svelte';
  import PurgeConfirm from '../components/library/PurgeConfirm.svelte';
  import Toolbar from '../components/library/Toolbar.svelte';
  import { refreshFacets } from '../lib/facets.svelte';
  import { formatBytes } from '../lib/format';
  import { Library } from '../lib/library.svelte';
  import { phone } from '../lib/media.svelte';
  import { isTypingTarget } from '../lib/modal';
  import { bottomBar } from '../lib/toast.svelte';
  import { onUploaded, uploadDialog } from '../lib/upload.svelte';

  let { mode }: { mode: 'library' | 'trash' } = $props();

  // The router remounts this screen when the mode changes, so the mode is fixed for one instance.
  const lib = new Library(untrack(() => mode));
  const trash = lib.mode === 'trash';

  let searchEl = $state<HTMLInputElement | null>(null);
  let filterSheet = $state(false);

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

  function onkeydown(e: KeyboardEvent) {
    if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey || isTypingTarget(e.target)) return;
    const overlays = document.querySelectorAll('[data-overlay]').length;
    const arrow = e.key === 'ArrowLeft' || e.key === 'ArrowRight';
    if (arrow) {
      // Arrows move through the images only while the detail drawer is the only thing open.
      if (lib.active && overlays === 1) {
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
    } else if (e.key === 'Escape' && lib.selecting) {
      lib.clearSel();
    }
  }

  const head = $derived(lib.facets);
  const retention = $derived(lib.retention);
</script>

<svelte:window {onkeydown} />

<div class="page">
  <div class="page-head">
    <div>
      <h1>{trash ? '回收站' : '全部图片'}</h1>
      {#if head}
        <p>
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
    {:else}
      <div class="page-actions">
        <button type="button" class="btn btn-primary" onclick={() => (uploadDialog.open = true)}>
          <Icon name="upload" />
          上传
        </button>
      </div>
    {/if}
  </div>

  <Toolbar {lib} bind:searchEl onopenfilters={() => (filterSheet = true)} />
  <ImageCollection {lib} />
</div>

{#if lib.active}
  <Drawer label="图片详情" width="440px" onclose={() => lib.close()}>
    <DetailPanel {lib} image={lib.active} onclose={() => lib.close()} />
  </Drawer>
{/if}

{#if filterSheet}
  <FilterSheet {lib} onclose={() => (filterSheet = false)} />
{/if}

{#if lib.purgeIds}
  <PurgeConfirm count={lib.purgeIds.length} oncancel={() => (lib.purgeIds = null)} onconfirm={() => lib.confirmPurge()} />
{/if}
