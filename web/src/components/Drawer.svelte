<script lang="ts">
  import type { Snippet } from 'svelte';
  import { createLeave } from '../lib/leave.svelte';
  import { overlay } from '../lib/modal';

  interface Props {
    label: string;
    onclose: () => void;
    width?: string;
    /** Receives `close`, which plays the exit animation before calling `onclose`. */
    children: Snippet<[() => void]>;
  }
  let { label, onclose, width, children }: Props = $props();

  const leave = createLeave(() => onclose, 200);
</script>

<div class="scrim" class:leaving={leave.leaving} role="presentation" onclick={leave.close}></div>
<div
  class="drawer"
  class:leaving={leave.leaving}
  role="dialog"
  aria-modal="true"
  aria-label={label}
  tabindex="-1"
  style:--w={width}
  use:overlay={{ onescape: leave.close }}
>
  {@render children(leave.close)}
</div>
