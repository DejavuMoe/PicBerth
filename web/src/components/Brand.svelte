<script lang="ts">
  import Icon from './Icon.svelte';

  let { href = '/library' }: { href?: string } = $props();
</script>

<a class="brand" {href} aria-label="PicBerth">
  <span class="brand-mark"><Icon name="image" size={14} /></span>
  <span class="brand-name">PicBerth</span>
</a>
