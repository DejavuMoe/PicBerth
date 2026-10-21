// Tiny History API router: exact paths only, query params, link interception, leave guards.

export const ROUTES = ['/', '/login', '/setup', '/library', '/trash', '/tokens', '/settings', '/status'] as const;

const state = $state({ path: normalize(location.pathname), search: location.search });

export const router = {
  get path() {
    return state.path;
  },
  get search() {
    return state.search;
  },
  get query(): URLSearchParams {
    return new URLSearchParams(state.search);
  },
};

function normalize(path: string): string {
  return path.length > 1 ? path.replace(/\/+$/, '') || '/' : path;
}

export function isKnownPath(path: string): boolean {
  return (ROUTES as readonly string[]).includes(normalize(path));
}

let lastHref = location.pathname + location.search;
function apply() {
  state.path = normalize(location.pathname);
  state.search = location.search;
  lastHref = location.pathname + location.search;
}

// A guard returns false (or resolves false) to veto leaving the current path.
type Guard = (target: string) => boolean | Promise<boolean>;
const guards = new Set<Guard>();

export function addGuard(g: Guard): () => void {
  guards.add(g);
  return () => guards.delete(g);
}

async function allowed(target: string): Promise<boolean> {
  for (const g of [...guards]) {
    if (!(await g(target))) return false;
  }
  return true;
}

/** Ask the leave guards without navigating (e.g. before signing out). */
export function canLeave(target: string): Promise<boolean> {
  return allowed(target);
}

export interface NavigateOptions {
  replace?: boolean;
  /** Skip leave guards (used after a confirmed discard). */
  force?: boolean;
}

export async function navigate(to: string, opts: NavigateOptions = {}): Promise<boolean> {
  const url = new URL(to, location.href);
  if (url.origin !== location.origin) {
    location.assign(url.href);
    return true;
  }
  const target = url.pathname + url.search + url.hash;
  const pathChanged = normalize(url.pathname) !== state.path;
  if (target === location.pathname + location.search + location.hash) return true;
  if (pathChanged && !opts.force && guards.size > 0 && !(await allowed(target))) return false;
  if (opts.replace) history.replaceState(null, '', target);
  else history.pushState(null, '', target);
  apply();
  if (pathChanged && !opts.replace) window.scrollTo(0, 0);
  return true;
}

/** Merge query parameters into the current URL without adding a history entry. */
export function setQuery(patch: Record<string, string | undefined>) {
  const p = new URLSearchParams(location.search);
  for (const [k, v] of Object.entries(patch)) {
    if (v === undefined || v === '') p.delete(k);
    else p.set(k, v);
  }
  const s = p.toString();
  history.replaceState(null, '', location.pathname + (s ? `?${s}` : ''));
  apply();
}

/** Only same-site app paths may be used as a post-login destination. */
export function safeNext(next: string | null): string {
  if (!next || !next.startsWith('/') || next.startsWith('//') || next.includes('\\')) return '/';
  try {
    const u = new URL(next, location.origin);
    if (u.origin !== location.origin || !isKnownPath(u.pathname)) return '/';
    if (u.pathname === '/login' || u.pathname === '/setup') return '/';
    return u.pathname + u.search;
  } catch {
    return '/';
  }
}

let started = false;
export function initRouter() {
  if (started) return;
  started = true;
  let bypass = false;

  window.addEventListener('popstate', async () => {
    if (bypass) {
      bypass = false;
      apply();
      return;
    }
    const target = location.pathname + location.search;
    if (normalize(location.pathname) !== state.path && guards.size > 0) {
      // Undo the browser's move, ask, and only then really go back.
      history.pushState(null, '', lastHref);
      if (await allowed(target)) {
        bypass = true;
        history.back();
      }
      return;
    }
    apply();
  });

  document.addEventListener('click', (e) => {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    const el = e.target instanceof Element ? e.target.closest('a[href]') : null;
    if (!(el instanceof HTMLAnchorElement)) return;
    if ((el.target && el.target !== '_self') || el.hasAttribute('download')) return;
    const url = new URL(el.href, location.href);
    if (url.origin !== location.origin || !isKnownPath(url.pathname)) return;
    e.preventDefault();
    void navigate(url.pathname + url.search + url.hash);
  });
}
