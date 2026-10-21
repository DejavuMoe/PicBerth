<script lang="ts">
  import type { Snippet } from 'svelte';
  import { overlay } from '../lib/modal';

  interface Props {
    label: string;
    onclose: () => void;
    children: Snippet;
  }
  let { label, onclose, children }: Props = $props();
</script>

<div class="scrim" role="presentation" onclick={onclose}></div>
<div class="sheet" role="dialog" aria-modal="true" aria-label={label} tabindex="-1" use:overlay={{ onescape: onclose }}>
  <div class="sheet-grip"></div>
  {@render children()}
</div>
