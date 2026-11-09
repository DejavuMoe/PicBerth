<script lang="ts">
  import { tick, untrack } from 'svelte';
  import type { Library } from '../../lib/library.svelte';
  import { phone, wideNav } from '../../lib/media.svelte';
  import { openPicker } from '../../lib/picker';
  import Check from '../Check.svelte';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import Breadcrumb from './Breadcrumb.svelte';
  import BulkActions from './BulkActions.svelte';
  import DisplayMenu from './DisplayMenu.svelte';
  import FilterPopover from './FilterPopover.svelte';
  import FolderPanel from './FolderPanel.svelte';

  interface Props {
    lib: Library;
    searchEl?: HTMLInputElement | null;
    searching?: boolean;
  }
  let { lib, searchEl = $bindable(null), searching = $bindable(false) }: Props = $props();

  const trash = $derived(lib.mode === 'trash');
  const items = $derived(lib.items);
  const allOn = $derived(items.length > 0 && items.every((i) => lib.sel.has(i.id)));
  const someOn = $derived(items.some((i) => lib.sel.has(i.id)));
  const count = $derived(
    lib.phase === 'ready' && (trash || lib.foldersPhase === 'ready') ? lib.countText : null,
  );

  // The box edits local text; the URL (and the query) follows after a short pause. Enter applies at once.
  let text = $state(untrack(() => lib.filters.q));
  $effect(() => {
    const v = text.trim();
    if (v === lib.filters.q) return;
    const t = setTimeout(() => lib.setFilters({ q: v }), 200);
    return () => clearTimeout(t);
  });
  $effect(() => {
    const q = lib.filters.q;
    untrack(() => {
      if (q !== text.trim()) text = q;
    });
  });
  $effect(() => {
    if (searching && phone.matches && searchEl) void tick().then(() => searchEl?.focus());
  });

  function toggleAll() {
    if (allOn) lib.selectNone();
    else lib.selectAll();
  }

  function cancelSearch() {
    text = '';
    lib.setFilters({ q: '' });
    searching = false;
  }

  function searchKey(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      lib.setFilters({ q: text.trim() });
    } else if (e.key === 'Escape') {
      e.stopPropagation();
      if (text) {
        text = '';
        lib.setFilters({ q: '' });
      } else if (e.currentTarget instanceof HTMLInputElement) {
        e.currentTarget.blur();
      }
    }
  }

</script>

{#snippet folderTool()}
  <Menu align="left" class="popover" label="文件夹">
    {#snippet trigger(open, toggle)}
      <button
        type="button"
        class="btn btn-ghost btn-icon"
        aria-label="文件夹"
        title="文件夹"
        aria-haspopup="dialog"
        aria-expanded={open}
        onclick={toggle}
      >
        <Icon name="folder" />
      </button>
    {/snippet}
    {#snippet children(close)}
      <FolderPanel {lib} currentId={lib.folderId} counts={lib.counts} compact {close} />
    {/snippet}
  </Menu>
{/snippet}

{#if phone.matches && searching}
  <div class="head">
    <div class="field-wrap grow">
      <Icon name="search" />
      <input
        bind:this={searchEl}
        class="input"
        type="search"
        placeholder="搜索"
        aria-label="搜索"
        autocomplete="off"
        spellcheck="false"
        value={text}
        oninput={(e) => (text = e.currentTarget.value)}
        onkeydown={searchKey}
      />
    </div>
    <button type="button" class="btn btn-ghost" onclick={cancelSearch}>取消</button>
  </div>
{:else if lib.selecting}
  <div class="head" style:padding-left={phone.matches ? '8px' : '12px'}>
    <div class="bulk">
      <Check on={allOn ? true : someOn ? 'mixed' : false} onchange={toggleAll} label="全选" />
      <span class="n">已选 {lib.sel.size} 张</span>
      {#if !phone.matches && lib.sel.size > 0}
        <BulkActions {lib} />
      {/if}
      <span class="grow"></span>
      <button type="button" class="btn btn-ghost" onclick={() => lib.clearSel()}>
        {phone.matches ? '完成' : '取消选择'}
      </button>
    </div>
  </div>
{:else}
  <div class="head">
    {#if trash}
      <div class="head-title">
        <h1>回收站</h1>
        {#if count !== null}<span class="count">{count}</span>{/if}
      </div>
    {:else}
      <Breadcrumb {lib} currentId={lib.folderId} {count} tool={!wideNav.matches ? folderTool : undefined} />
    {/if}
    {#if phone.matches}
      <button
        type="button"
        class="btn btn-ghost btn-icon"
        aria-label="搜索"
        title="搜索"
        onclick={() => (searching = true)}
      >
        <Icon name="search" />
      </button>
    {:else}
      <div class="field-wrap search">
        <Icon name="search" />
        <input
          bind:this={searchEl}
          class="input"
          type="search"
          placeholder="搜索"
          aria-label="搜索"
          autocomplete="off"
          spellcheck="false"
          value={text}
          oninput={(e) => (text = e.currentTarget.value)}
          onkeydown={searchKey}
        />
        {#if !text}<kbd>/</kbd>{/if}
      </div>
    {/if}
    <FilterPopover {lib} />
    <DisplayMenu {lib} />
    {#if !trash}
      <button
        type="button"
        class="btn btn-primary"
        class:btn-icon={phone.matches}
        style:color={phone.matches ? 'var(--accent-ink)' : undefined}
        aria-label="上传"
        title="上传 (U)"
        onclick={openPicker}
      >
        <Icon name="upload" />
        {#if !phone.matches}上传{/if}
      </button>
    {/if}
  </div>
{/if}
