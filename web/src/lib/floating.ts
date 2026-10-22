// Floating layers (menus, selects, popovers): drawn into <body> so no scroll container, overflow clip or
// transformed overlay can cut them off, and placed beside their anchor.

export interface PlaceOptions {
  anchor: HTMLElement | null;
  /** Which edge of the anchor the layer lines up with. */
  align?: 'left' | 'right';
  /** Prefer opening above the anchor. */
  up?: boolean;
  /** At least as wide as the anchor. */
  matchWidth?: boolean;
  minWidth?: number | undefined;
}

/** Distance kept from the viewport edge, and the gap between anchor and layer. */
const MARGIN = 8;
const GAP = 4;

/** Svelte action: moves the node to <body> (Svelte has no portal). */
export function portal(node: HTMLElement) {
  document.body.appendChild(node);
  return {
    destroy() {
      node.remove();
    },
  };
}

/**
 * Svelte action: places the node beside its anchor — start edge, 4px below, above when there is no room below,
 * shifted to stay 8px inside the viewport — and keeps it there on scroll, resize and content changes.
 * Sets `data-placed` once the first position is known (the stylesheet hides the layer until then).
 */
export function place(node: HTMLElement, initial: PlaceOptions) {
  let o = initial;
  let frame = 0;

  function position() {
    const anchor = o.anchor;
    if (!anchor || !anchor.isConnected) return;
    const r = anchor.getBoundingClientRect();
    const vw = document.documentElement.clientWidth;
    const vh = document.documentElement.clientHeight;
    const min = o.matchWidth ? r.width : o.minWidth;
    node.style.minWidth = min ? `${min}px` : '';
    node.style.maxHeight = '';
    const w = node.offsetWidth;
    const h = node.offsetHeight;

    const below = vh - r.bottom - GAP - MARGIN;
    const above = r.top - GAP - MARGIN;
    const preferred = o.up ? 'up' : 'down';
    const fits = (dir: 'up' | 'down') => (dir === 'up' ? above : below) >= h;
    const other = preferred === 'up' ? 'down' : 'up';
    const dir = fits(preferred) ? preferred : fits(other) ? other : above > below ? 'up' : 'down';
    const room = dir === 'up' ? above : below;

    const start = o.align === 'right' ? r.right - w : r.left;
    node.style.left = `${Math.max(MARGIN, Math.min(start, vw - w - MARGIN))}px`;
    if (dir === 'up') {
      node.style.top = '';
      node.style.bottom = `${vh - r.top + GAP}px`;
    } else {
      node.style.bottom = '';
      node.style.top = `${r.bottom + GAP}px`;
    }
    node.style.maxHeight = h > room ? `${Math.max(120, room)}px` : '';
    node.dataset.placed = '';
  }

  function schedule() {
    if (frame) return;
    frame = requestAnimationFrame(() => {
      frame = 0;
      position();
    });
  }
  // A scroll inside the layer itself (a long option list) does not move it.
  function onscroll(e: Event) {
    if (e.target instanceof Node && node.contains(e.target)) return;
    schedule();
  }

  position();
  window.addEventListener('resize', schedule);
  window.addEventListener('scroll', onscroll, { capture: true, passive: true });
  const ro = new ResizeObserver(schedule);
  ro.observe(node);

  return {
    update(next: PlaceOptions) {
      o = next;
      position();
    },
    destroy() {
      cancelAnimationFrame(frame);
      window.removeEventListener('resize', schedule);
      window.removeEventListener('scroll', onscroll, { capture: true });
      ro.disconnect();
    },
  };
}
