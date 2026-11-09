// Reactive media queries. The listeners live for the page's lifetime, so create these once at module level.

function watch(query: string) {
  const mq = typeof matchMedia === 'function' ? matchMedia(query) : null;
  const state = $state({ matches: mq?.matches ?? false });
  mq?.addEventListener('change', (e) => (state.matches = e.matches));
  return state;
}

/** Phone layout: tab bar, list rows, filter sheet, fixed bulk bar. */
export const phone = watch('(max-width: 720px)');

/** Tablet layout: the sidebar collapses to an icon rail. */
export const tablet = watch('(min-width: 721px) and (max-width: 1099px)');

/** Wide layout: room for the folder tree pane beside the list. */
export const wideNav = watch('(min-width: 1100px)');

/** Wide layout: room for the tree pane, the list and a docked 360px detail column. */
export const wide = watch('(min-width: 1440px)');
