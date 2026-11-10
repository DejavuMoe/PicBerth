<script lang="ts">
  import { SvelteSet } from 'svelte/reactivity';
  import { ROOT_LABEL } from '../../lib/folders';
  import { tree } from '../../lib/tree.svelte';
  import type { Folder } from '../../lib/types';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import FolderMenu from './FolderMenu.svelte';

  interface Props {
    /** Inside the tablet/phone popover: the tree has its own header and scrolls inside a floating panel. */
    compact?: boolean;
    /** Closes the popover after a folder is opened. */
    close?: () => void;
  }
  let { compact = false, close }: Props = $props();

  /** The folder on screen; undefined (nothing marked) on other screens. */
  const currentId = $derived(tree.current);
  const counts = $derived(tree.counts);

  const DRAG_TYPE = 'application/x-picberth-images';

  type Editing = { mode: 'create'; parent: string | null } | { mode: 'rename'; id: string };

  const open = new SvelteSet<string>();
  let editing = $state<Editing | null>(null);
  let draft = $state('');
  let err = $state('');
  let over = $state<string | null>(null);
  let inputEl = $state<HTMLInputElement | null>(null);
  let busy = false;

  const editKey = $derived(
    editing ? (editing.mode === 'rename' ? `r${editing.id}` : `c${editing.parent ?? 'root'}`) : '',
  );

  $effect(() => {
    for (const f of tree.chain(currentId ?? null)) open.add(f.id);
  });

  // The palette's 新建文件夹 starts a new root folder.
  let lastSignal = tree.createSignal;
  $effect(() => {
    const n = tree.createSignal;
    if (n === lastSignal) return;
    lastSignal = n;
    startCreate(null);
  });

  $effect(() => {
    const key = editKey;
    const el = inputEl;
    if (!key || !el) return;
    el.focus();
    el.select();
  });

  // Hovering a collapsed node during a drag expands it after a moment.
  $effect(() => {
    const id = over;
    if (!id || id === 'root' || open.has(id) || tree.kids(id).length === 0) return;
    const t = setTimeout(() => open.add(id), 700);
    return () => clearTimeout(t);
  });

  function cancelEdit() {
    editing = null;
    err = '';
  }

  function startCreate(parent: string | null) {
    if (tree.phase !== 'ready') return;
    editing = { mode: 'create', parent };
    draft = '';
    err = '';
    if (parent) open.add(parent);
  }

  function startRename(f: Folder) {
    editing = { mode: 'rename', id: f.id };
    draft = f.name;
    err = '';
  }

  async function commit(fromBlur: boolean) {
    const edit = editing;
    if (!edit || busy) return;
    const name = draft.trim();
    if (edit.mode === 'rename') {
      const cur = tree.folders.find((f) => f.id === edit.id);
      if (!cur) return cancelEdit();
      if (!name && fromBlur) return cancelEdit();
      if (cur.name === name) return cancelEdit();
    } else if (!name && fromBlur) {
      return cancelEdit();
    }
    busy = true;
    const e = edit.mode === 'rename' ? await tree.renameFolder(edit.id, name) : await tree.createFolder(edit.parent, name);
    busy = false;
    if (!e || fromBlur) cancelEdit();
    else err = e;
  }

  interface Entry {
    kind: 'folder' | 'editor';
    level: number;
    key: string;
    folder?: Folder;
  }

  const entries = $derived.by(() => {
    const out: Entry[] = [];
    const walk = (parent: string | null, level: number) => {
      for (const f of tree.kids(parent)) {
        if (editing?.mode === 'rename' && editing.id === f.id) {
          out.push({ kind: 'editor', level, key: `r${f.id}` });
          continue;
        }
        out.push({ kind: 'folder', level, key: f.id, folder: f });
        if (open.has(f.id)) {
          walk(f.id, level + 1);
          if (editing?.mode === 'create' && editing.parent === f.id) {
            out.push({ kind: 'editor', level: level + 1, key: `c${f.id}` });
          }
        }
      }
    };
    walk(null, 1);
    if (editing?.mode === 'create' && editing.parent === null) {
      out.push({ kind: 'editor', level: 1, key: 'croot' });
    }
    return out;
  });

  function toggle(id: string) {
    if (open.has(id)) open.delete(id);
    else open.add(id);
  }

  function dropProps(id: string | null) {
    const key = id ?? 'root';
    return {
      ondragover: (e: DragEvent) => {
        if (!e.dataTransfer?.types.includes(DRAG_TYPE)) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        if (over !== key) over = key;
      },
      ondragleave: (e: DragEvent) => {
        if (!(e.currentTarget as Node).contains(e.relatedTarget as Node | null)) over = null;
      },
      ondrop: (e: DragEvent) => {
        const raw = e.dataTransfer?.getData(DRAG_TYPE);
        over = null;
        if (!raw) return;
        e.preventDefault();
        try {
          const ids = JSON.parse(raw) as unknown;
          if (Array.isArray(ids)) tree.dropImages(ids.filter((x): x is string => typeof x === 'string'), id);
        } catch {
          // not our payload
        }
      },
    };
  }
</script>

{#snippet addButton(cls: string)}
  <button
    type="button"
    class="btn btn-ghost btn-icon btn-sm {cls}"
    aria-label="新建文件夹"
    title="新建文件夹"
    disabled={tree.phase !== 'ready'}
    onclick={() => startCreate(null)}
  >
    <Icon name="folder-plus" />
  </button>
{/snippet}

{#snippet editor(level: number, key: string)}
  <div class="tn tn-edit" style:padding-left={`${8 + (level - 1) * 16}px`}>
    <span class="tn-tw"></span>
    <Icon name="folder" />
    <input
      bind:this={inputEl}
      class="tn-input"
      aria-label="文件夹名称"
      aria-invalid={err ? 'true' : undefined}
      value={draft}
      maxlength={80}
      oninput={(e) => {
        draft = e.currentTarget.value;
        err = '';
      }}
      onblur={() => void commit(true)}
      onkeydown={(e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          void commit(false);
        } else if (e.key === 'Escape') {
          e.stopPropagation();
          cancelEdit();
        }
      }}
    />
  </div>
  {#if err}
    <div class="field-error tn-err" role="alert" style:padding-left={`${8 + (level - 1) * 16 + 44}px`}>{err}</div>
  {/if}
{/snippet}

{#snippet rootRow()}
  <div class="tn tn-root" class:is-cur={currentId === null} class:is-over={over === 'root'} {...dropProps(null)}>
    <button
      type="button"
      class="tn-main"
      aria-current={currentId === null ? 'page' : undefined}
      onclick={() => {
        tree.openFolder(null);
        close?.();
      }}
    >
      <Icon name="image" />
      <span class="ellipsis">{ROOT_LABEL}</span>
      {#if tree.phase === 'ready' && (counts.get(null) ?? 0) > 0}<span class="n num">{counts.get(null)}</span>{/if}
    </button>
    {#if !compact}{@render addButton('tn-more')}{/if}
  </div>
{/snippet}

{#if compact}
  <div class="tree-head">
    <span class="medium">文件夹</span>
    {@render addButton('')}
  </div>
{/if}
<nav class="tree-body" aria-label="文件夹">
  {@render rootRow()}
  {#if tree.phase === 'loading'}
    <div aria-busy="true" aria-label="加载中" class="tree-skel">
      {#each [60, 40, 52, 36, 48] as w, i (i)}
        <div class="skeleton" style:width={`${w}%`} style:height="10px" style:margin-left={`${28 + (i % 3 === 2 ? 16 : 0)}px`}></div>
      {/each}
    </div>
  {:else if tree.phase === 'error'}
    <div class="tree-note" role="alert">
      <span>文件夹加载失败</span>
      <button type="button" class="btn btn-secondary btn-sm" onclick={() => tree.load()}>重试</button>
    </div>
  {:else}
    {#each entries as entry (entry.key)}
      {#if entry.kind === 'editor'}
        {@render editor(entry.level, entry.key)}
      {:else if entry.folder}
        {@const f = entry.folder}
        {@const kids = tree.kids(f.id)}
        {@const expanded = open.has(f.id)}
        {@const n = counts.get(f.id) ?? 0}
        <div
          class="tn"
          class:is-cur={currentId === f.id}
          class:is-over={over === f.id}
          style:padding-left={`${8 + (entry.level - 1) * 16}px`}
          {...dropProps(f.id)}
        >
          <button
            type="button"
            class="tn-tw"
            aria-label={expanded ? `收起 ${f.name}` : `展开 ${f.name}`}
            aria-expanded={kids.length ? expanded : undefined}
            disabled={kids.length === 0}
            tabindex={kids.length ? 0 : -1}
            onclick={() => toggle(f.id)}
          >
            {#if kids.length > 0}<Icon name={expanded ? 'down' : 'right'} size="s" />{/if}
          </button>
          <button
            type="button"
            class="tn-main"
            aria-current={currentId === f.id ? 'page' : undefined}
            onclick={() => {
              tree.openFolder(f.id);
              close?.();
            }}
          >
            <Icon name="folder" />
            <span class="ellipsis">{f.name}</span>
            {#if n > 0}<span class="n num">{n}</span>{/if}
          </button>
          <Menu align="right" nested={compact}>
            {#snippet trigger(openMenu, toggleMenu)}
              <button
                type="button"
                class="btn btn-ghost btn-icon btn-sm tn-more"
                aria-label={`${f.name} 的更多操作`}
                title="更多"
                aria-haspopup="menu"
                aria-expanded={openMenu}
                onclick={toggleMenu}
              >
                <Icon name="more" />
              </button>
            {/snippet}
            {#snippet children(closeMenu)}
              <FolderMenu
                folder={f}
                nonEmpty={kids.length > 0 || n > 0}
                onnew={() => startCreate(f.id)}
                onrename={() => startRename(f)}
                ondelete={() => void tree.deleteFolder(f.id)}
                close={closeMenu}
              />
            {/snippet}
          </Menu>
        </div>
      {/if}
    {/each}
    {#if tree.kids(null).length === 0 && !editing}
      <div class="tree-note">
        <span>还没有文件夹</span>
        <button type="button" class="btn btn-ghost btn-sm" onclick={() => startCreate(null)}>新建</button>
      </div>
    {/if}
  {/if}
</nav>
