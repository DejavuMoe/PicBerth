<script lang="ts">
  import type { Snippet } from 'svelte';
  import { createLeave } from '../lib/leave.svelte';
  import { overlay } from '../lib/modal';
  import Icon from './Icon.svelte';

  interface Props {
    title: string;
    onclose: () => void;
    width?: string;
    /** Phones show it full screen. */
    fullOnPhone?: boolean;
    /** false: neither Escape nor a click outside closes it (the close button still does). */
    dismissible?: boolean;
    /** Dropped or pasted files are ignored while this is open. */
    blocksDrop?: boolean;
    /** Both snippets receive `close`, which plays the exit animation before calling `onclose`. */
    children: Snippet<[() => void]>;
    footer?: Snippet<[() => void]>;
  }
  let { title, onclose, width, fullOnPhone = false, dismissible = true, blocksDrop = true, children, footer }: Props = $props();

  const leave = createLeave(() => onclose, 140);
</script>

<div class="scrim modal-scrim" class:leaving={leave.leaving} role="presentation" onclick={() => dismissible && leave.close()}></div>
<div
  class="modal"
  class:full-sm={fullOnPhone}
  class:leaving={leave.leaving}
  role="dialog"
  aria-modal="true"
  aria-label={title}
  tabindex="-1"
  style:--w={width}
  use:overlay={{ onescape: dismissible ? leave.close : undefined, blocksDrop }}
>
  <div class="modal-head">
    <h2>{title}</h2>
    <button type="button" class="btn btn-ghost btn-icon btn-sm" onclick={leave.close} aria-label="关闭"><Icon name="x" /></button>
  </div>
  <div class="modal-body">{@render children(leave.close)}</div>
  {#if footer}<div class="modal-foot">{@render footer(leave.close)}</div>{/if}
</div>
