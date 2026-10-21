// Shared behaviour for overlay surfaces (modal, drawer, sheet): a stack so only the top one reacts to
// Escape, a Tab focus trap, scroll lock and focus restore.

let locks = 0;

/** Lock page scroll while an overlay is open; returns the release function. */
export function lockScroll(): () => void {
  if (locks++ === 0) document.documentElement.classList.add('scroll-locked');
  let released = false;
  return () => {
    if (released) return;
    released = true;
    if (--locks === 0) document.documentElement.classList.remove('scroll-locked');
  };
}

interface Entry {
  el: HTMLElement;
  onescape?: () => void;
}
const stack: Entry[] = [];

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

function focusables(el: HTMLElement): HTMLElement[] {
  return Array.from(el.querySelectorAll<HTMLElement>(FOCUSABLE)).filter((n) => n.offsetParent !== null || n === document.activeElement);
}

function onkeydown(e: KeyboardEvent) {
  const top = stack[stack.length - 1];
  if (!top || e.defaultPrevented) return;
  if (e.key === 'Escape' && top.onescape) {
    e.preventDefault();
    top.onescape();
  } else if (e.key === 'Tab') {
    const items = focusables(top.el);
    const first = items[0];
    const last = items[items.length - 1];
    if (!first || !last) {
      e.preventDefault();
      top.el.focus();
    } else if (!top.el.contains(document.activeElement)) {
      e.preventDefault();
      first.focus();
    } else if (e.shiftKey && (document.activeElement === first || document.activeElement === top.el)) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }
}

export interface OverlayOptions {
  /** Called for Escape when this is the top-most overlay; omit to ignore Escape. */
  onescape?: (() => void) | undefined;
  /** Dropped or pasted files are ignored while a blocking overlay is open. */
  blocksDrop?: boolean;
}

/** Svelte action for the root element of an overlay. */
export function overlay(node: HTMLElement, opts: OverlayOptions) {
  const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  const entry: Entry = { el: node, onescape: opts.onescape };
  node.dataset.overlay = '';
  if (opts.blocksDrop) node.dataset.blocksDrop = '';
  if (stack.length === 0) document.addEventListener('keydown', onkeydown);
  stack.push(entry);
  const unlock = lockScroll();
  queueMicrotask(() => {
    if (!node.isConnected || node.contains(document.activeElement)) return;
    (node.querySelector<HTMLElement>('[data-autofocus]') ?? node).focus({ preventScroll: true });
  });
  return {
    update(next: OverlayOptions) {
      entry.onescape = next.onescape;
    },
    destroy() {
      const i = stack.indexOf(entry);
      if (i >= 0) stack.splice(i, 1);
      if (stack.length === 0) document.removeEventListener('keydown', onkeydown);
      unlock();
      if (previous?.isConnected) previous.focus({ preventScroll: true });
    },
  };
}

export function isTypingTarget(t: EventTarget | null): boolean {
  if (!(t instanceof HTMLElement)) return false;
  if (t.isContentEditable) return true;
  const tag = t.tagName;
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT';
}

export function modalOpen(): boolean {
  return document.querySelector('[data-overlay]') !== null;
}

export function dropBlocked(): boolean {
  return document.querySelector('[data-overlay][data-blocks-drop]') !== null;
}
