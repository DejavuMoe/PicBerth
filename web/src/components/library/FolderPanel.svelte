<script lang="ts">
  import { SvelteSet } from 'svelte/reactivity';
  import { ROOT_LABEL } from '../../lib/folders';
  import type { Library } from '../../lib/library.svelte';
  import type { Folder } from '../../lib/types';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import FolderMenu from './FolderMenu.svelte';

  interface Props {
    lib: Library;
    currentId: string | null;
    counts: Map<string | null, number>;
    /** Inside the tablet/phone popover: the tree scrolls inside a floating panel. */
    compact?: boolean;
    /** Closes the popover after a folder is opened. */
    close?: () => void;
  }
  let { lib, currentId, counts, compact = false, close }: Props = $props();

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
    for (const f of lib.chain(currentId)) open.add(f.id);
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
    if (!id || id === 'root' || open.has(id) || lib.kids(id).length === 0) return;
    const t = setTimeout(() => open.add(id), 700);
    return () => clearTimeout(t);
  });

  function cancelEdit() {
    editing = null;
    err = '';
  }

  function startCreate(parent: string | null) {
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
      const cur = lib.folders.find((f) => f.id === edit.id);
      if (!cur) return cancelEdit();
      if (!name && fromBlur) return cancelEdit();
      if (cur.name === name) return cancelEdit();
    } else if (!name && fromBlur) {
      return cancelEdit();
    }
    busy = true;
    const e = edit.mode === 'rename' ? await lib.renameFolder(edit.id, name) : await lib.createFolder(edit.parent, name);
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
      for (const f of lib.kids(parent)) {
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
          if (Array.isArray(ids)) void lib.moveImages(ids.filter((x): x is string => typeof x === 'string'), id);
        } catch {
          // not our payload
        }
      },
    };
  }
</script>

<div class="tree-head">
  <span class="medium">文件夹</span>
  <button
    type="button"
    class="btn btn-ghost btn-icon btn-sm"
    aria-label="新建文件夹"
    title="新建文件夹"
    disabled={lib.foldersPhase !== 'ready'}
    onclick={() => startCreate(null)}
  >
    <Icon name="folder-plus" />
  </button>
</div>
<div class="tree-body">
  {#if lib.foldersPhase === 'loading'}
    <div aria-busy="true" aria-label="加载中" class="tree-skel">
      {#each [60, 40, 52, 36, 48] as w, i (i)}
        <div class="skeleton" style:width={`${w}%`} style:height="10px" style:margin-left={`${8 + (i % 3 === 2 ? 16 : 0)}px`}></div>
      {/each}
    </div>
  {:else if lib.foldersPhase === 'error'}
    <div class="tree-note" role="alert">
      <span>文件夹加载失败</span>
      <button type="button" class="btn btn-secondary btn-sm" onclick={() => lib.loadFolders()}>重试</button>
    </div>
  {:else}
    <nav aria-label="文件夹">
      <div class="tn" class:is-cur={currentId === null} class:is-over={over === 'root'} style:padding-left="4px" {...dropProps(null)}>
        <span class="tn-tw"></span>
        <button
          type="button"
          class="tn-main"
          aria-current={currentId === null ? 'page' : undefined}
          onclick={() => {
            lib.openFolder(null);
            close?.();
          }}
        >
          <Icon name="image" />
          <span class="ellipsis">{ROOT_LABEL}</span>
          {#if (counts.get(null) ?? 0) > 0}<span class="n num">{counts.get(null)}</span>{/if}
        </button>
      </div>

      {#each entries as entry (entry.key)}
        {#if entry.kind === 'editor'}
          <div class="tn tn-edit" style:padding-left={`${4 + (entry.level - 1) * 16}px`}>
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
            <div class="field-error tn-err" role="alert" style:padding-left={`${4 + (entry.level - 1) * 16 + 44}px`}>{err}</div>
          {/if}
        {:else if entry.folder}
          {@const f = entry.folder}
          {@const kids = lib.kids(f.id)}
          {@const expanded = open.has(f.id)}
          {@const n = counts.get(f.id) ?? 0}
          <div
            class="tn"
            class:is-cur={currentId === f.id}
            class:is-over={over === f.id}
            style:padding-left={`${4 + (entry.level - 1) * 16}px`}
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
                lib.openFolder(f.id);
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
                  {lib}
                  folder={f}
                  nonEmpty={kids.length > 0 || n > 0}
                  onnew={() => startCreate(f.id)}
                  onrename={() => startRename(f)}
                  ondelete={() => void lib.deleteFolder(f.id)}
                  close={closeMenu}
                />
              {/snippet}
            </Menu>
          </div>
        {/if}
      {/each}

      {#if lib.kids(null).length === 0 && !editing}
        <div class="tree-note muted">还没有文件夹</div>
      {/if}
    </nav>
  {/if}
</div>
