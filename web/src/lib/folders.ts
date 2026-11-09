// Pure helpers for the folder tree: names, depth, paths and validation. No state.

import type { Folder } from './types';

export const ROOT_LABEL = '图片';
export const MAX_DEPTH = 3;

const byName = (a: Folder, b: Folder) => a.name.localeCompare(b.name, 'zh-CN');

export function childrenOf(folders: readonly Folder[], parentId: string | null): Folder[] {
  return folders.filter((f) => (f.parentId ?? null) === parentId).sort(byName);
}

export function folderById(folders: readonly Folder[], id: string | null): Folder | undefined {
  return id === null ? undefined : folders.find((f) => f.id === id);
}

/** Root first, then each ancestor down to the folder itself. */
export function folderChain(folders: readonly Folder[], id: string | null): Folder[] {
  const out: Folder[] = [];
  const seen = new Set<string>();
  let f = folderById(folders, id);
  while (f && !seen.has(f.id)) {
    out.unshift(f);
    seen.add(f.id);
    f = folderById(folders, f.parentId ?? null);
  }
  return out;
}

export function folderDepth(folders: readonly Folder[], id: string | null): number {
  return id === null ? 0 : folderChain(folders, id).length;
}

/** 图片 / 旅行 / 2026. */
export function folderLabel(folders: readonly Folder[], id: string | null): string {
  return [ROOT_LABEL, ...folderChain(folders, id).map((f) => f.name)].join(' / ');
}

/** Folder names below the root, without the 图片 prefix. */
export function folderNames(folders: readonly Folder[], id: string | null): string[] {
  return folderChain(folders, id).map((f) => f.name);
}

export function descendantIds(folders: readonly Folder[], id: string): Set<string> {
  const out = new Set<string>();
  const walk = (parent: string) => {
    for (const f of childrenOf(folders, parent)) {
      out.add(f.id);
      walk(f.id);
    }
  };
  walk(id);
  return out;
}

/** How many levels this folder's own subtree spans (a leaf is 1). */
export function subtreeHeight(folders: readonly Folder[], id: string): number {
  const kids = childrenOf(folders, id);
  return 1 + (kids.length ? Math.max(...kids.map((k) => subtreeHeight(folders, k.id))) : 0);
}

/** Direct images in a folder plus everything below it. */
export function subtreeCount(folders: readonly Folder[], id: string): number {
  let n = folderById(folders, id)?.imageCount ?? 0;
  for (const d of descendantIds(folders, id)) n += folderById(folders, d)?.imageCount ?? 0;
  return n;
}

/** Moving `id` under `targetId` (null = root) must keep the deepest descendant at depth <= 3. */
export function fitsUnder(folders: readonly Folder[], id: string, targetId: string | null): boolean {
  return folderDepth(folders, targetId) + subtreeHeight(folders, id) <= MAX_DEPTH;
}

/** Names -> folder id. null = root, undefined = no such folder. */
export function resolvePath(folders: readonly Folder[], names: readonly string[]): string | null | undefined {
  let parent: string | null = null;
  for (const name of names) {
    const f: Folder | undefined = childrenOf(folders, parent).find((x) => x.name === name);
    if (!f) return undefined;
    parent = f.id;
  }
  return parent;
}

/** Inside means the folder itself or one of its descendants. */
export function isInside(folders: readonly Folder[], ancestorId: string, id: string | null): boolean {
  if (id === null) return false;
  return id === ancestorId || descendantIds(folders, ancestorId).has(id);
}

export function folderNameError(
  folders: readonly Folder[],
  parentId: string | null,
  name: string,
  selfId?: string | null,
): string {
  const n = name.trim();
  if (!n) return '请输入名称';
  if (n.length > 64) return '名称最多 64 个字符';
  if (/[\\/]/.test(n)) return '名称不能包含 / 或 \\';
  if (childrenOf(folders, parentId ?? null).some((f) => f.name === n && f.id !== selfId)) return '已有同名文件夹';
  return '';
}
