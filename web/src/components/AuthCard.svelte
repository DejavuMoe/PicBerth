<script lang="ts">
  import type { Snippet } from 'svelte';
  import BrandMark from './BrandMark.svelte';

  let { title, children }: { title: string; children: Snippet } = $props();
</script>

<main class="auth">
  <div class="auth-col">
    <span role="img" aria-label="PicBerth"><BrandMark large /></span>
    <h1>{title}</h1>
    {@render children()}
  </div>
</main>
