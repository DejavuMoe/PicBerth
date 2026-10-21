<script lang="ts">
  import type { Visibility } from '../lib/types';
  import Icon from './Icon.svelte';

  let { visibility }: { visibility: Visibility } = $props();
</script>

{#if visibility === 'public'}
  <span class="badge badge-success"><span class="dot"></span>公开</span>
{:else}
  <span class="badge"><Icon name="lock" size={12} />私有</span>
{/if}
