<script lang="ts">
  import type { Snippet } from 'svelte';

  /** A small muted heading above a stack of hairline rows. `compact` rows are 40px instead of 44px. */
  let { title, compact = false, children }: { title: string; compact?: boolean; children: Snippet } = $props();
</script>

<section class="adm-sec" class:adm-compact={compact}>
  <h2>{title}</h2>
  <div class="adm-rows">{@render children()}</div>
</section>
