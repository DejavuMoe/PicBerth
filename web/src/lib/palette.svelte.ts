// The command palette (Ctrl/⌘ K) and the requests it makes of other screens.

export const palette = $state({ open: false, newToken: false });

export const IS_MAC = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
/** The modifier shown in hints. */
export const MOD = IS_MAC ? '⌘' : 'Ctrl';

export function openPalette() {
  palette.open = true;
}

export function closePalette() {
  palette.open = false;
}
