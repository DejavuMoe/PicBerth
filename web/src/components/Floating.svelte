<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';
  import { place, portal } from '../lib/floating';

  interface Props extends Omit<HTMLAttributes<HTMLDivElement>, 'class' | 'children'> {
    /** The element the layer is placed beside; a press on it does not count as "outside". */
    anchor: HTMLElement | null;
    /** Called for Escape and for a press outside the layer. */
    onclose: () => void;
    align?: 'left' | 'right';
    up?: boolean;
    matchWidth?: boolean;
    minWidth?: number | undefined;
    /** The floating-layer class: `menu` or `popover`. */
    class?: string;
    /** A layer opened from inside another popover: presses here must not dismiss the outer one. */
    nested?: boolean;
    el?: HTMLElement | null;
    children: Snippet;
  }
  let {
    anchor,
    onclose,
    align = 'left',
    up = false,
    matchWidth = false,
    minWidth,
    class: className = 'menu',
    nested = false,
    el = $bindable(null),
    children,
    ...rest
  }: Props = $props();

  // Escape is taken in the capture phase and stopped, so a layer opened over a drawer or modal closes alone.
  $effect(() => {
    const layer = el;
    if (!layer) return;
    const down = (e: PointerEvent) => {
      const t = e.target;
      if (!(t instanceof Node) || layer.contains(t) || anchor?.contains(t)) return;
      // A menu opened from inside this popover lives in its own layer.
      if (t instanceof Element && t.closest('[data-nested]')) return;
      onclose();
    };
    const key = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      e.preventDefault();
      e.stopImmediatePropagation();
      onclose();
    };
    document.addEventListener('pointerdown', down, true);
    document.addEventListener('keydown', key, true);
    return () => {
      document.removeEventListener('pointerdown', down, true);
      document.removeEventListener('keydown', key, true);
    };
  });
</script>

<div
  bind:this={el}
  class={className}
  use:portal
  use:place={{ anchor, align, up, matchWidth, minWidth }}
  data-float
  data-nested={nested ? 'true' : undefined}
  {...rest}
>
  {@render children()}
</div>
