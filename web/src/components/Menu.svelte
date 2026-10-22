<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    trigger: Snippet<[boolean, () => void]>;
    children: Snippet<[() => void]>;
    align?: 'left' | 'right';
    up?: boolean;
    width?: number;
    /** Extra class on the popover (e.g. `menu-facets`). */
    class?: string;
  }
  let { trigger, children, align = 'left', up = false, width, class: className = '' }: Props = $props();

  let open = $state(false);
  let wrap = $state<HTMLElement | null>(null);
  const toggle = () => (open = !open);
  const close = () => (open = false);

  // Escape is taken in the capture phase so a menu inside a drawer closes alone.
  $effect(() => {
    if (!open || !wrap) return;
    const el = wrap;
    const down = (e: PointerEvent) => {
      if (e.target instanceof Node && !el.contains(e.target)) close();
    };
    const key = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        e.stopPropagation();
        close();
        el.querySelector<HTMLElement>('[aria-expanded]')?.focus();
        return;
      }
      if ((e.key !== 'ArrowDown' && e.key !== 'ArrowUp') || !el.contains(document.activeElement)) return;
      const items = Array.from(el.querySelectorAll<HTMLElement>('.menu [role^="menuitem"]'));
      if (items.length === 0) return;
      e.preventDefault();
      const i = items.indexOf(document.activeElement as HTMLElement);
      const next = e.key === 'ArrowDown' ? (i + 1) % items.length : (i <= 0 ? items.length : i) - 1;
      items[next]?.focus();
    };
    document.addEventListener('pointerdown', down);
    document.addEventListener('keydown', key, true);
    return () => {
      document.removeEventListener('pointerdown', down);
      document.removeEventListener('keydown', key, true);
    };
  });
</script>

<div class="menu-wrap" bind:this={wrap}>
  {@render trigger(open, toggle)}
  {#if open}
    <div class="menu {className}" class:right={align === 'right'} class:up role="menu" style:min-width={width ? `${width}px` : undefined}>
      {@render children(close)}
    </div>
  {/if}
</div>
