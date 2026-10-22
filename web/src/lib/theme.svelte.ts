// Theme choice: follow the system (default), light or dark. An explicit light/dark choice lives in
// localStorage; "system" is stored as no value. The choice is written to <html data-theme>, where the
// stylesheet resolves it (and sets color-scheme); the browser theme-color follows the resulting canvas colour.

export type ThemeChoice = 'system' | 'light' | 'dark';
export type Theme = 'light' | 'dark';

export const THEME_OPTIONS: { key: ThemeChoice; label: string }[] = [
  { key: 'system', label: '跟随系统' },
  { key: 'light', label: '浅色' },
  { key: 'dark', label: '深色' },
];

const KEY = 'picberth.theme';
const media = typeof matchMedia === 'function' ? matchMedia('(prefers-color-scheme: dark)') : null;

function stored(): ThemeChoice {
  try {
    const v = localStorage.getItem(KEY);
    return v === 'light' || v === 'dark' ? v : 'system';
  } catch {
    return 'system';
  }
}

const state = $state({ choice: stored(), system: (media?.matches ? 'dark' : 'light') as Theme });

export const theme = {
  /** What the user picked: system, light or dark. */
  get choice(): ThemeChoice {
    return state.choice;
  },
  /** The theme actually in effect. */
  get resolved(): Theme {
    return state.choice === 'system' ? state.system : state.choice;
  },
};

function syncMeta() {
  const color = getComputedStyle(document.documentElement).getPropertyValue('--bg').trim();
  for (const m of document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')) {
    m.content = color || m.content;
  }
}

/** Apply the current choice to the document (called from main.ts before the first paint). */
export function applyTheme() {
  document.documentElement.dataset.theme = state.choice;
  syncMeta();
}

media?.addEventListener('change', (e) => {
  state.system = e.matches ? 'dark' : 'light';
  if (state.choice === 'system') applyTheme();
});

export function setTheme(choice: ThemeChoice) {
  state.choice = choice;
  try {
    if (choice === 'system') localStorage.removeItem(KEY);
    else localStorage.setItem(KEY, choice);
  } catch {
    // storage unavailable: the choice lasts for this page only
  }
  applyTheme();
}
