<script lang="ts" generics="T">
  // Lays a sequence of frames out as film strips: one strip per row, the last strip only as long as
  // its frames (docs/ui.md §5.1). The caller renders each frame; this component only cuts the film.
  import type { Snippet } from 'svelte';

  interface Props {
    items: T[];
    /** Frames per strip. */
    cols: number;
    key: (item: T) => string | number;
    frame: Snippet<[T, number]>;
    /** Extra class on every strip (e.g. a muted trash style). */
    tone?: string;
  }
  let { items, cols, key, frame, tone = '' }: Props = $props();

  const strips = $derived.by(() => {
    const n = Math.max(1, cols);
    const out: { start: number; items: T[] }[] = [];
    for (let i = 0; i < items.length; i += n) out.push({ start: i, items: items.slice(i, i + n) });
    return out;
  });
</script>

<div class="roll" style:--cols={Math.max(1, cols)}>
  {#each strips as strip (strip.start)}
    <div class="strip film {tone}" style:--n={strip.items.length}>
      {#each strip.items as item, j (key(item))}
        {@render frame(item, strip.start + j)}
      {/each}
    </div>
  {/each}
</div>

<style>
  .roll {
    --gap: 6px;
    --pad: 8px; /* = the .film side padding */
    container-type: inline-size;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .strip {
    /* Frame width comes from the full roll width, so a short last strip is simply shorter. */
    --fw: calc((100cqw - 2 * var(--pad) - (var(--cols) - 1) * var(--gap)) / var(--cols));
    display: grid;
    grid-template-columns: repeat(var(--n), var(--fw));
    gap: var(--gap);
    width: max-content;
    max-width: 100%;
    border-radius: 1px;
  }
</style>
