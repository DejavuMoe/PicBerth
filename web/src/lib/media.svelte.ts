// Reactive media queries. The listeners live for the page's lifetime, so create these once at module level.

function watch(query: string) {
  const mq = typeof matchMedia === 'function' ? matchMedia(query) : null;
  const state = $state({ matches: mq?.matches ?? false });
  mq?.addEventListener('change', (e) => (state.matches = e.matches));
  return state;
}

/** Phone layout: list rows, filter sheet, fixed bulk bar. */
export const phone = watch('(max-width: 720px)');
