<script lang="ts">
  import type { Snippet } from 'svelte';
  import { createLeave } from '../lib/leave.svelte';
  import { overlay } from '../lib/modal';

  interface Props {
    title: string;
    onclose: () => void;
    /** The 560px dialog. */
    wide?: boolean;
    /** false: neither Escape nor a click outside closes it. */
    dismissible?: boolean;
    /** Dropped or pasted files are ignored while this is open. */
    blocksDrop?: boolean;
    /** Snippet receives `close`, which plays the exit animation before calling `onclose`. */
    children: Snippet<[() => void]>;
    footer?: Snippet<[() => void]>;
  }
  let { title, onclose, wide = false, dismissible = true, blocksDrop = true, children, footer }: Props = $props();

  const leave = createLeave(() => onclose, 250);
</script>

<div
  class="scrim"
  class:leaving={leave.leaving}
  role="presentation"
  onpointerdown={(e) => dismissible && e.target === e.currentTarget && leave.close()}
>
  <div
    class="dialog"
    class:dialog-wide={wide}
    class:leaving={leave.leaving}
    role="dialog"
    aria-modal="true"
    aria-labelledby="dlg-title"
    use:overlay={{ onescape: dismissible ? leave.close : undefined, blocksDrop }}
  >
    <h2 class="dialog-head" id="dlg-title">{title}</h2>
    <div class="dialog-body">{@render children(leave.close)}</div>
    {#if footer}<div class="dialog-foot">{@render footer(leave.close)}</div>{/if}
  </div>
</div>

<style>
  /* Exit animations; entry animations come from the global .scrim/.dialog rules. */
  .scrim.leaving {
    animation: m-scrim-out var(--dur) var(--ease) forwards;
  }
  .dialog.leaving {
    animation: m-dialog-out var(--dur) var(--ease) forwards;
  }
  .scrim.leaving,
  .dialog.leaving {
    pointer-events: none;
  }
  @keyframes m-scrim-out {
    to {
      opacity: 0;
    }
  }
  @keyframes m-dialog-out {
    to {
      opacity: 0;
      transform: translateY(4px);
    }
  }
</style>
