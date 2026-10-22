<script lang="ts">
  import type { Visibility } from '../lib/types';
  import Icon from './Icon.svelte';

  /** The padlock that marks a private image; public images show nothing. */
  let { visibility }: { visibility: Visibility } = $props();
</script>

{#if visibility === 'private'}
  <span class="lock-mark" role="img" aria-label="私有" title="私有"><Icon name="lock" size={13} /></span>
{/if}
