<script lang="ts">
  import type { Snippet } from 'svelte';
  import { createLeave } from '../lib/leave.svelte';
  import { overlay } from '../lib/modal';

  interface Props {
    /** Accessible name of the dialog. */
    label: string;
    onclose: () => void;
    /** Receives `close`, which plays the exit animation before calling `onclose`. */
    children: Snippet<[() => void]>;
  }
  let { label, onclose, children }: Props = $props();

  const leave = createLeave(() => onclose, 250);
</script>

<!-- The library screen renders `aside.detail.drawer` inside; this component is overlay + behaviour only. -->
<div class="detail-overlay" class:leaving={leave.leaving} role="presentation" onclick={leave.close}></div>
<div class="detail-host" role="dialog" aria-modal="true" aria-label={label} use:overlay={{ onescape: leave.close }}>
  {@render children(leave.close)}
</div>

<style>
  /* display: contents: the slot child (aside.detail.drawer) keeps the prototype's fixed positioning. */
  .detail-host {
    display: contents;
  }
  .detail-overlay.leaving {
    animation: overlay-out var(--dur) var(--ease) forwards;
    pointer-events: none;
  }
  @keyframes overlay-out {
    to {
      opacity: 0;
    }
  }
</style>
