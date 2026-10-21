<script lang="ts">
  import type { Snippet } from 'svelte';
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
    children: Snippet;
    footer?: Snippet;
  }
  let { title, onclose, width, fullOnPhone = false, dismissible = true, blocksDrop = true, children, footer }: Props = $props();
</script>

<div class="scrim modal-scrim" role="presentation" onclick={() => dismissible && onclose()}></div>
<div
  class="modal"
  class:full-sm={fullOnPhone}
  role="dialog"
  aria-modal="true"
  aria-label={title}
  tabindex="-1"
  style:--w={width}
  use:overlay={{ onescape: dismissible ? onclose : undefined, blocksDrop }}
>
  <div class="modal-head">
    <h2>{title}</h2>
    <button type="button" class="btn btn-ghost btn-icon btn-sm" onclick={onclose} aria-label="关闭"><Icon name="x" /></button>
  </div>
  <div class="modal-body">{@render children()}</div>
  {#if footer}<div class="modal-foot">{@render footer()}</div>{/if}
</div>
