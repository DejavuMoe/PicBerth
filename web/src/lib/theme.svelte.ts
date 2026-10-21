// 亮台 (light) / 暗房 (dark). The explicit choice lives in localStorage; without one the system decides.

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

media?.addEventListener('change', (e) => {
  state.system = e.matches ? 'dark' : 'light';
  syncMeta();
});

export const theme = {
  get current(): Theme {
    return state.choice ?? state.system;
  },
};

function syncMeta() {
  const color = getComputedStyle(document.documentElement).getPropertyValue('--bg').trim();
  for (const m of document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')) {
    m.content = color || m.content;
  }
}

/** Apply the stored choice before the first paint (called from main.ts). */
export function applyTheme() {
  if (state.choice) document.documentElement.dataset.theme = state.choice;
  else delete document.documentElement.dataset.theme;
  syncMeta();
}

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
