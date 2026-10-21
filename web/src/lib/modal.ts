// Shared helpers for native <dialog> based surfaces (Dialog, Drawer).

let locks = 0;

/** Lock page scroll while a modal surface is open; returns the release function. */
export function lockScroll(): () => void {
  if (locks++ === 0) {
    // Hand the reserved scrollbar gutter to the body as padding, so the modal's backdrop can cover
    // the whole window while the page behind it does not move.
    const html = document.documentElement;
    // html.clientWidth ignores a reserved-but-empty gutter in Chrome; the root box's width does not.
    const gutter = window.innerWidth - html.getBoundingClientRect().width;
    html.style.setProperty('--locked-gutter', `${Math.max(0, gutter)}px`);
    html.classList.add('scroll-locked');
  }
  let released = false;
  return () => {
    if (released) return;
    released = true;
    if (--locks === 0) document.documentElement.classList.remove('scroll-locked');
  };
}

/**
 * Close on a click that both starts and ends on the <dialog> element itself, i.e. on the ::backdrop
 * (the content box fills the dialog). A text selection dragged out of the box does not count.
 */
export function closeOnBackdrop(dialog: HTMLDialogElement, close: () => void): () => void {
  let downOnBackdrop = false;
  const down = (e: PointerEvent) => {
    downOnBackdrop = e.target === dialog;
  };
  const click = (e: MouseEvent) => {
    if (e.target === dialog && downOnBackdrop) close();
    downOnBackdrop = false;
  };
  dialog.addEventListener('pointerdown', down);
  dialog.addEventListener('click', click);
  return () => {
    dialog.removeEventListener('pointerdown', down);
    dialog.removeEventListener('click', click);
  };
}

export function isTypingTarget(t: EventTarget | null): boolean {
  if (!(t instanceof HTMLElement)) return false;
  if (t.isContentEditable) return true;
  const tag = t.tagName;
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT';
}

export function modalOpen(): boolean {
  return document.querySelector('dialog[open]') !== null;
}
