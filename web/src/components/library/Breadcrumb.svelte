<script lang="ts">
  import type { Snippet } from 'svelte';
  import { ROOT_LABEL } from '../../lib/folders';
  import type { Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';

  interface Props {
    lib: Library;
    currentId: string | null;
    count: string | null;
    tool?: Snippet;
  }
  let { lib, currentId, count, tool }: Props = $props();

  const DRAG_TYPE = 'application/x-picberth-images';
  const chain = $derived(lib.chain(currentId));
  const last = $derived(chain[chain.length - 1]);
  let over = $state<string | null>(null);

  function dropProps(id: string | null) {
    const key = id ?? 'root';
    return {
      ondragover: (e: DragEvent) => {
        if (!e.dataTransfer?.types.includes(DRAG_TYPE)) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        if (over !== key) over = key;
      },
      ondragleave: () => {
        if (over === key) over = null;
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

<div class="head-title">
  {#if last}
    <nav class="crumb-trail" aria-label="位置">
      <span class="crumb-anc">
        <button type="button" class="crumb" class:is-over={over === 'root'} onclick={() => lib.openFolder(null)} {...dropProps(null)}>
          {ROOT_LABEL}
        </button>
        <span class="sep" aria-hidden="true"><Icon name="right" size="s" /></span>
      </span>
      {#each chain.slice(0, -1) as f (f.id)}
        <span class="crumb-anc">
          <button type="button" class="crumb" class:is-over={over === f.id} onclick={() => lib.openFolder(f.id)} {...dropProps(f.id)}>
            {f.name}
          </button>
          <span class="sep" aria-hidden="true"><Icon name="right" size="s" /></span>
        </span>
      {/each}
      <h1 class="ellipsis">{last.name}</h1>
    </nav>
  {:else}
    <h1>{ROOT_LABEL}</h1>
  {/if}
  {#if count !== null}<span class="count">{count}</span>{/if}
  {#if tool}<span class="crumb-tool">{@render tool()}</span>{/if}
</div>
