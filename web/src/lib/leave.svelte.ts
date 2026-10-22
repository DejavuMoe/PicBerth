// Exit animation for overlays: close() plays the `.leaving` animation, then calls the real onclose.
// With reduced motion the overlay closes at once.

import { onDestroy } from 'svelte';

const reduced = typeof matchMedia === 'function' ? matchMedia('(prefers-reduced-motion: reduce)') : null;

/** Call during component init; `getClose` is read at close time so a changed `onclose` prop is honoured. */
export function createLeave(getClose: () => () => void, ms: number) {
  let leaving = $state(false);
  let closed = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  onDestroy(() => clearTimeout(timer));
  return {
    get leaving() {
      return leaving;
    },
    close() {
      if (closed) return;
      closed = true;
      if (reduced?.matches) {
        getClose()();
        return;
      }
      leaving = true;
      timer = setTimeout(() => getClose()(), ms);
    },
  };
}
