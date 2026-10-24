<script lang="ts">
  import type { Snippet } from 'svelte';
  import { createLeave } from '../lib/leave.svelte';
  import { overlay } from '../lib/modal';

  interface Props {
    label: string;
    onclose: () => void;
    /** Fills the screen below a small top margin. */
    full?: boolean;
    /** Receives `close`, which plays the exit animation before calling `onclose`. */
    children: Snippet<[() => void]>;
  }
  let { label, onclose, full = false, children }: Props = $props();

  const leave = createLeave(() => onclose, 250);
</script>

<div class="scrim" class:leaving={leave.leaving} role="presentation" onclick={leave.close}></div>
<div
  class="sheet"
  class:sheet-full={full}
  class:leaving={leave.leaving}
  role="dialog"
  aria-modal="true"
  aria-label={label}
  tabindex="-1"
  use:overlay={{ onescape: leave.close }}
>
  <div class="sheet-grip"></div>
  {@render children(leave.close)}
</div>
