<script lang="ts">
  import type { Visibility } from '../lib/types';
  import Icon from './Icon.svelte';

  let { visibility }: { visibility: Visibility } = $props();
</script>

{#if visibility === 'public'}
  <span class="vis"><Icon name="globe" size={13} />公开</span>
{:else}
  <span class="vis vis-private"><Icon name="lock" size={13} />私有</span>
{/if}
