<script lang="ts">
  import { ROOT_LABEL } from '../../lib/folders';
  import { tree } from '../../lib/tree.svelte';
  import type { Folder } from '../../lib/types';
  import MenuItem from '../MenuItem.svelte';

  interface Props {
    /** Omitted or null ticks the root; null means the root. */
    currentId?: string | null;
    /** Folders that may not be picked (the folder itself and its subtree). */
    exclude?: Set<string>;
    /** Why a target cannot be picked; empty means it can. */
    blocked?: (id: string) => string;
    onpick: (id: string | null) => void;
    close: () => void;
  }
  let { currentId, exclude, blocked, onpick, close }: Props = $props();

  interface Row {
    folder: Folder;
    level: number;
  }
  const rows = $derived.by(() => {
    const out: Row[] = [];
    const walk = (parent: string | null, level: number) => {
      for (const f of tree.kids(parent)) {
        if (exclude?.has(f.id)) continue;
        out.push({ folder: f, level });
        walk(f.id, level + 1);
      }
    };
    walk(null, 1);
    return out;
  });

  function pick(id: string | null) {
    close();
    if (id !== currentId) onpick(id);
  }
</script>

<MenuItem checked={currentId === null} onclick={() => pick(null)}>{ROOT_LABEL}</MenuItem>
{#each rows as row (row.folder.id)}
  {@const reason = blocked?.(row.folder.id) ?? ''}
  <MenuItem
    indent={row.level + 1}
    checked={currentId === row.folder.id}
    disabled={!!reason}
    end={reason || undefined}
    onclick={() => pick(row.folder.id)}
  >
    {row.folder.name}
  </MenuItem>
{/each}
