<script lang="ts">
  import { tick, type Snippet } from 'svelte';
  import Floating from './Floating.svelte';

  interface Props {
    /** Renders the button that opens the menu; `open` is for aria-expanded, `toggle` for its click. */
    trigger: Snippet<[boolean, () => void]>;
    /** The menu content; `close` closes the menu and returns focus to the trigger. */
    children: Snippet<[() => void]>;
    /** Which edge of the trigger the menu lines up with. */
    align?: 'left' | 'right';
    /** Prefer opening above the trigger. */
    up?: boolean;
    /** Minimum width of the menu in px. */
    width?: number;
    /** Extra class on the menu element (e.g. `popover`). */
    class?: string;
    /** Accessible name of the menu. */
    label?: string;
    /** Opened from inside a popover: presses in the menu must not dismiss that popover. */
    nested?: boolean;
  }
  let { trigger, children, align = 'left', up = false, width, class: className = 'menu', label, nested = false }: Props = $props();

  let open = $state(false);
  let wrap = $state<HTMLElement | null>(null);
  let layer = $state<HTMLElement | null>(null);
  /** The menu was opened from the keyboard: move focus into it. */
  let byKeyboard = false;

  const toggle = () => (open = !open);
  const items = () => (layer ? Array.from(layer.querySelectorAll<HTMLElement>('[role^="menuitem"]:not(:disabled)')) : []);

  function close() {
    if (layer?.contains(document.activeElement)) {
      (wrap?.querySelector<HTMLElement>('[aria-expanded]') ?? wrap?.querySelector<HTMLElement>('button'))?.focus();
    }
    open = false;
  }
  /** Escape and outside presses also hand focus back to the trigger, as in the prototype. */
  function dismiss() {
    (wrap?.querySelector<HTMLElement>('[aria-expanded]') ?? wrap?.querySelector<HTMLElement>('button'))?.focus();
    close();
  }

  $effect(() => {
    if (!layer || !byKeyboard) return;
    byKeyboard = false;
    (layer.querySelector<HTMLElement>('[aria-checked="true"]') ?? items()[0])?.focus();
  });

  function focusItem(at: number) {
    const list = items();
    list[(at + list.length) % list.length]?.focus();
  }

  function onwrapkeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      if (!open) {
        byKeyboard = true;
        open = true;
        void tick();
      } else focusItem(e.key === 'ArrowDown' ? 0 : -1);
    } else if (e.key === 'Enter' || e.key === ' ') {
      byKeyboard = true;
    }
  }

  function onlayerkeydown(e: KeyboardEvent) {
    const list = items();
    if (e.key === 'Tab') {
      close();
      return;
    }
    if (list.length === 0) return;
    const i = list.indexOf(document.activeElement as HTMLElement);
    if (e.key === 'ArrowDown') focusItem(i + 1);
    else if (e.key === 'ArrowUp') focusItem(i < 0 ? -1 : i - 1);
    else if (e.key === 'Home') focusItem(0);
    else if (e.key === 'End') focusItem(-1);
    else return;
    e.preventDefault();
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="menu-wrap" bind:this={wrap} onkeydown={onwrapkeydown} onpointerdown={() => (byKeyboard = false)}>
  {@render trigger(open, toggle)}
</div>
{#if open}
  <Floating
    anchor={wrap}
    bind:el={layer}
    onclose={dismiss}
    {align}
    {up}
    minWidth={width}
    class={className}
    role={className === 'popover' ? 'dialog' : 'menu'}
    aria-label={label}
    {nested}
    onkeydown={onlayerkeydown}
  >
    {@render children(close)}
  </Floating>
{/if}
