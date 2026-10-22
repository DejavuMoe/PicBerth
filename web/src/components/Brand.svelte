<script lang="ts">
  import BrandMark from './BrandMark.svelte';

  let { href = '/library' }: { href?: string } = $props();
</script>

<a class="brand" {href} aria-label="PicBerth">
  <BrandMark />
  <span class="brand-name">PicBerth</span>
</a>
