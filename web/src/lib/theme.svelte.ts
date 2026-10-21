// Light / dark. The explicit choice lives in localStorage; without one the system decides.
// The resolved theme is always written to <html data-theme>.

export type Theme = 'light' | 'dark';

const KEY = 'picberth.theme';
const media = typeof matchMedia === 'function' ? matchMedia('(prefers-color-scheme: dark)') : null;

function stored(): Theme | null {
  try {
    const v = localStorage.getItem(KEY);
    return v === 'light' || v === 'dark' ? v : null;
  } catch {
    return null;
  }
}

const state = $state({ choice: stored(), system: (media?.matches ? 'dark' : 'light') as Theme });

export const theme = {
  get current(): Theme {
    return state.choice ?? state.system;
  },
};

function syncMeta() {
  const color = getComputedStyle(document.documentElement).getPropertyValue('--canvas').trim();
  for (const m of document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')) {
    m.content = color || m.content;
  }
}

/** Apply the current theme to the document (called from main.ts before the first paint). */
export function applyTheme() {
  document.documentElement.dataset.theme = theme.current;
  syncMeta();
}

media?.addEventListener('change', (e) => {
  state.system = e.matches ? 'dark' : 'light';
  applyTheme();
});

export function toggleTheme() {
  const next: Theme = theme.current === 'dark' ? 'light' : 'dark';
  // Returning to the system's own preference clears the override.
  state.choice = next === state.system ? null : next;
  try {
    if (state.choice) localStorage.setItem(KEY, state.choice);
    else localStorage.removeItem(KEY);
  } catch {
    // storage unavailable: the choice lasts for this page only
  }
  applyTheme();
}
