// Counts for the filter menus, the sidebar and the page head. Refreshed after uploads and mutations.

import { api } from './api';
import type { Facets, ImageState } from './types';

export const facets = $state({ active: null as Facets | null, trashed: null as Facets | null });

let inflight: Promise<void> | null = null;
let again = false;

async function load() {
  const [a, t] = await Promise.allSettled([api.facets('active'), api.facets('trashed')]);
  if (a.status === 'fulfilled') facets.active = a.value;
  if (t.status === 'fulfilled') facets.trashed = t.value;
}

export function refreshFacets(): Promise<void> {
  if (inflight) {
    again = true;
    return inflight;
  }
  inflight = load().finally(() => {
    inflight = null;
    if (again) {
      again = false;
      void refreshFacets();
    }
  });
  return inflight;
}

export function facetsFor(state: ImageState): Facets | null {
  return state === 'trashed' ? facets.trashed : facets.active;
}
