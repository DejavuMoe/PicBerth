// The folder tree, shared by the sidebar, the library screen and the palette. One instance for the whole
// signed-in session; the current folder always comes from the path.

import { api, ApiError, errorMessage } from './api';
import { facetsFor } from './facets.svelte';
import {
  MAX_DEPTH,
  childrenOf,
  fitsUnder,
  folderById,
  folderChain,
  folderDepth,
  folderLabel,
  folderNameError,
  folderNames,
  isInside,
  resolvePath,
} from './folders';
import { isLibraryPath, libraryPath, librarySegments, navigate, router } from './router.svelte';
import { toast } from './toast.svelte';
import type { Folder } from './types';

/** Map a folder request failure onto the inline message the tree shows. */
function folderError(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.code === 'folder-exists') return '已有同名文件夹';
    if (e.code === 'folder-depth') return `最多 ${MAX_DEPTH} 层`;
    if (e.code === 'folder-not-empty') return '文件夹不为空';
  }
  return errorMessage(e);
}

class FolderTree {
  folders = $state<Folder[]>([]);
  phase = $state<'loading' | 'ready' | 'error'>('loading');
  error = $state('');
  /** Counts up when something outside the tree (the palette) asks it to start a new root folder. */
  createSignal = $state(0);
  /** Set by the library screen: moves dropped images (the tree itself does not know the image list). */
  mover: ((ids: string[], to: string | null) => void) | null = null;

  #inflight: Promise<void> | null = null;

  /** The folder id on screen: null at the library root, undefined on other screens or for an unknown folder. */
  current = $derived.by((): string | null | undefined => {
    const p = router.path;
    if (p !== '/' && !isLibraryPath(p)) return undefined;
    if (this.phase !== 'ready') return null;
    return resolvePath(this.folders, librarySegments(p));
  });

  load(quiet = false): Promise<void> {
    if (this.#inflight) return this.#inflight;
    if (!quiet || this.phase !== 'ready') this.phase = 'loading';
    this.#inflight = (async () => {
      try {
        const page = await api.listFolders();
        this.folders = page.items;
        this.phase = 'ready';
        this.error = '';
      } catch (e) {
        this.error = errorMessage(e);
        if (this.phase !== 'ready') this.phase = 'error';
      } finally {
        this.#inflight = null;
      }
    })();
    return this.#inflight;
  }

  /** Load on first use, refresh quietly afterwards. */
  ensure(): Promise<void> {
    return this.load(this.phase === 'ready');
  }

  refresh(): Promise<void> {
    return this.load(true);
  }

  reset() {
    this.folders = [];
    this.phase = 'loading';
    this.error = '';
    this.mover = null;
  }

  // ---- reading -----------------------------------------------------------------------------------

  label(id: string | null): string {
    return folderLabel(this.folders, id);
  }

  names(id: string | null): string[] {
    return folderNames(this.folders, id);
  }

  chain(id: string | null): Folder[] {
    return folderChain(this.folders, id);
  }

  kids(id: string | null): Folder[] {
    return childrenOf(this.folders, id);
  }

  /** Direct image counts for the tree, including the root's own count. */
  get counts(): Map<string | null, number> {
    const m = new Map<string | null, number>();
    for (const f of this.folders) m.set(f.id, f.imageCount);
    m.set(null, this.rootCount);
    return m;
  }

  get rootCount(): number {
    const total = facetsFor('active')?.total;
    if (total === undefined) return 0;
    return Math.max(0, total - this.folders.reduce((n, f) => n + f.imageCount, 0));
  }

  // ---- navigation --------------------------------------------------------------------------------

  openFolder(id: string | null) {
    // Filters and sort live in the query and stay with the folder when the library is already open.
    const keep = isLibraryPath(router.path) ? router.search : '';
    void navigate(libraryPath(folderNames(this.folders, id)) + keep);
  }

  dropImages(ids: string[], to: string | null) {
    this.mover?.(ids, to);
  }

  // ---- changes -----------------------------------------------------------------------------------

  async createFolder(parentId: string | null, name: string): Promise<string> {
    const e = folderNameError(this.folders, parentId, name);
    if (e) return e;
    if (folderDepth(this.folders, parentId) >= MAX_DEPTH) return `最多 ${MAX_DEPTH} 层`;
    try {
      await api.createFolder({ name: name.trim(), parentId });
      await this.load(true);
      toast('已创建文件夹');
      return '';
    } catch (err) {
      return folderError(err);
    }
  }

  async renameFolder(id: string, name: string): Promise<string> {
    const f = folderById(this.folders, id);
    if (!f) return '文件夹不存在';
    const e = folderNameError(this.folders, f.parentId ?? null, name, id);
    if (e) return e;
    const currentId = this.current ?? null;
    const here = isInside(this.folders, id, currentId);
    const search = router.search;
    try {
      await api.patchFolder(id, { name: name.trim() });
      await this.load(true);
      if (here) void navigate(libraryPath(folderNames(this.folders, currentId)) + search, { replace: true });
      toast('已重命名');
      return '';
    } catch (err) {
      return folderError(err);
    }
  }

  async moveFolder(id: string, to: string | null) {
    const f = folderById(this.folders, id);
    if (!f) return;
    if ((f.parentId ?? null) === to) return;
    if (folderNameError(this.folders, to, f.name, id)) return toast('目标位置已有同名文件夹');
    if (!fitsUnder(this.folders, id, to)) return toast(`最多 ${MAX_DEPTH} 层`);
    const before = f.parentId ?? null;
    const currentId = this.current ?? null;
    const here = isInside(this.folders, id, currentId);
    const search = router.search;
    const where = folderLabel(this.folders, to);
    try {
      await api.patchFolder(id, { parentId: to });
      await this.load(true);
      if (here) void navigate(libraryPath(folderNames(this.folders, currentId)) + search, { replace: true });
      toast(`已将 ${f.name} 移到 ${where}`, {
        action: {
          label: '撤销',
          run: () => {
            void (async () => {
              try {
                await api.patchFolder(id, { parentId: before });
                await this.load(true);
                if (here) void navigate(libraryPath(folderNames(this.folders, currentId)) + router.search, { replace: true });
              } catch (e) {
                toast(folderError(e));
              }
            })();
          },
        },
      });
    } catch (e) {
      toast(folderError(e));
    }
  }

  async deleteFolder(id: string) {
    const f = folderById(this.folders, id);
    if (!f) return;
    const nonEmpty = childrenOf(this.folders, id).length > 0 || (this.counts.get(id) ?? 0) > 0;
    if (nonEmpty) return toast('文件夹不为空');
    const parentId = f.parentId ?? null;
    const name = f.name;
    const leaving = this.current === id;
    const search = router.search;
    try {
      await api.deleteFolder(id);
      await this.load(true);
      if (leaving) void navigate(libraryPath(folderNames(this.folders, parentId)) + search, { replace: true });
      toast('已删除文件夹', {
        action: {
          label: '撤销',
          run: () => {
            void (async () => {
              try {
                await api.createFolder({ name, parentId });
                await this.load(true);
                if (leaving) {
                  void navigate(libraryPath([...folderNames(this.folders, parentId), name]) + router.search, { replace: true });
                }
              } catch (e) {
                toast(folderError(e));
              }
            })();
          },
        },
      });
    } catch (e) {
      toast(folderError(e));
    }
  }
}

export const tree = new FolderTree();
