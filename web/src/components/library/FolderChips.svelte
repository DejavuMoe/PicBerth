<script lang="ts">
  import type { Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';

  let { lib }: { lib: Library } = $props();

  const DRAG_TYPE = 'application/x-picberth-images';
  const list = $derived(lib.subfolders);
  let over = $state<string | null>(null);

  function dropProps(id: string) {
    return {
      ondragover: (e: DragEvent) => {
        if (!e.dataTransfer?.types.includes(DRAG_TYPE)) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        if (over !== id) over = id;
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

<!-- The direct subfolders of the current folder, above its images in the grid view. Each accepts dropped images. -->
{#if list.length > 0}
  <div class="fgrid" role="group" aria-label="文件夹">
    {#each list as f (f.id)}
      {@const n = lib.counts.get(f.id) ?? 0}
      <button
        type="button"
        class="fchip"
        class:is-over={over === f.id}
        onclick={() => lib.openFolder(f.id)}
        {...dropProps(f.id)}
      >
        <Icon name="folder" />
        <span class="ellipsis">{f.name}</span>
        {#if n > 0}<span class="n num">{n}</span>{/if}
      </button>
    {/each}
  </div>
{/if}
