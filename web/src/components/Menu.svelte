<script lang="ts">
  import type { Snippet } from 'svelte';
  import Floating from './Floating.svelte';

  interface Props {
    /** Renders the button that opens the menu; `open` is for aria-expanded, `toggle` for its click. */
    trigger: Snippet<[boolean, () => void]>;
    /** The menu content; `close` closes the menu and returns focus to the trigger. */
    children: Snippet<[() => void]>;
    align?: 'left' | 'right';
    /** Prefer opening above the trigger. */
    up?: boolean;
    /** Minimum width of the menu in px. */
    width?: number;
    /** At least as wide as the trigger. */
    matchWidth?: boolean;
    /** Extra class on the menu (e.g. `menu-facets`). */
    class?: string;
    /** Extra class on the wrapper around the trigger (e.g. `block`, `account-wrap`). */
    wrapClass?: string;
    label?: string;
  }
  let {
    trigger,
    children,
    align = 'left',
    up = false,
    width,
    matchWidth = false,
    class: className = '',
    wrapClass = '',
    label,
  }: Props = $props();

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
<div class="menu-wrap {wrapClass}" bind:this={wrap} onkeydown={onwrapkeydown} onpointerdown={() => (byKeyboard = false)}>
  {@render trigger(open, toggle)}
</div>
{#if open}
  <Floating
    anchor={wrap}
    bind:el={layer}
    onclose={close}
    {align}
    {up}
    {matchWidth}
    minWidth={width}
    class="menu {className}"
    role="menu"
    aria-label={label}
    onkeydown={onlayerkeydown}
  >
    {@render children(close)}
  </Floating>
{/if}
