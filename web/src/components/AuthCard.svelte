<script lang="ts">
  import type { Snippet } from 'svelte';
  import Brand from './Brand.svelte';

  let { title, children }: { title: string; children: Snippet } = $props();
</script>

<main class="auth">
  <div class="auth-card">
    <Brand href="/" />
    <h1>{title}</h1>
    {@render children()}
  </div>
</main>
