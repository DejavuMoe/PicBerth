<script lang="ts">
  import type { Snippet } from 'svelte';

  /** The 48px header at the top of the content panel: title on the left, the screen's controls (children) on the right.
   *  On phones it sticks to the top of the page. Use it as the first child of `<div class="page">`, followed by
   *  `<div class="page-body">` for the scrolling content. */
  let { title, children }: { title: string; children?: Snippet } = $props();
</script>

<header class="page-head">
  <h1>{title}</h1>
  {#if children}<div class="page-actions">{@render children()}</div>{/if}
</header>
