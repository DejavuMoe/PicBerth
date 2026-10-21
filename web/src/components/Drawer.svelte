<script lang="ts">
  import type { Snippet } from 'svelte';
  import { overlay } from '../lib/modal';

  interface Props {
    label: string;
    onclose: () => void;
    side?: 'left' | 'right';
    width?: string;
    children: Snippet;
  }
  let { label, onclose, side = 'right', width, children }: Props = $props();
</script>

<div class="scrim" role="presentation" onclick={onclose}></div>
<div
  class="drawer"
  class:left={side === 'left'}
  role="dialog"
  aria-modal="true"
  aria-label={label}
  tabindex="-1"
  style:--w={width}
  use:overlay={{ onescape: onclose }}
>
  {@render children()}
</div>
