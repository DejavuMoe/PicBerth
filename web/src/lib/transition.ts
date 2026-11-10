// The preview grows into the viewer (and back) with the View Transitions API where the browser has it and
// motion is allowed; everywhere else the change is immediate and the layers fade in on their own.

import { flushSync } from 'svelte';

const reduced = typeof matchMedia === 'function' ? matchMedia('(prefers-reduced-motion: reduce)') : null;

type Transitioning = Document & { startViewTransition?: (update: () => void) => unknown };

/** Run `change` (a state update) inside a view transition when one is available. */
export function withTransition(change: () => void) {
  const doc = document as Transitioning;
  if (!doc.startViewTransition || reduced?.matches) {
    change();
    return;
  }
  doc.startViewTransition(() => flushSync(change));
}
