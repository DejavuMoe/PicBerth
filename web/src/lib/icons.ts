// One hand-drawn icon set. 16 px viewBox, 1.5 px stroke, round caps and joins, currentColor.
// Strokes stay inside a 2 px safe margin. Filled marks are limited to tiny dots.

export interface IconShape {
  /** SVG path data, stroked. */
  d: string[];
  /** Tiny filled dots: [cx, cy, r]. */
  dots?: [number, number, number][];
}

const rect = (x: number, y: number, w: number, h: number, r: number) =>
  `M${x + r} ${y}h${w - 2 * r}a${r} ${r} 0 0 1 ${r} ${r}v${h - 2 * r}a${r} ${r} 0 0 1-${r} ${r}h-${w - 2 * r}a${r} ${r} 0 0 1-${r}-${r}v-${h - 2 * r}a${r} ${r} 0 0 1 ${r}-${r}z`;

export const ICONS = {
  // Brand: a square outline moored on a pier bar slightly wider than the square.
  mark: { d: [rect(3.75, 1.75, 8.5, 8.5, 2), 'M1.75 13.75h12.5'] },
  upload: { d: ['M8 10.25V2.75', 'M4.75 5.75 8 2.5l3.25 3.25', 'M2.75 10.5v2.75h10.5V10.5'] },
  download: { d: ['M8 2.5v7.75', 'M4.75 7 8 10.25 11.25 7', 'M2.75 10.5v2.75h10.5V10.5'] },
  copy: {
    d: [
      rect(5.75, 5.75, 7.5, 7.5, 1.25),
      'M5.75 10.25H4A1.25 1.25 0 0 1 2.75 9V4A1.25 1.25 0 0 1 4 2.75h5A1.25 1.25 0 0 1 10.25 4v1.75',
    ],
  },
  check: { d: ['M3 8.5 6.5 12 13 4.5'] },
  lock: { d: [rect(3, 7.25, 10, 6.25, 1.25), 'M5.25 7.25v-2a2.75 2.75 0 0 1 5.5 0v2'], dots: [[8, 10.4, 0.8]] },
  globe: {
    d: [
      'M8 2.25a5.75 5.75 0 1 0 0 11.5 5.75 5.75 0 0 0 0-11.5z',
      'M8 2.25c-2.3 1.7-2.3 9.8 0 11.5 2.3-1.7 2.3-9.8 0-11.5z',
      'M2.5 8h11',
    ],
  },
  trash: {
    d: [
      'M2.75 4.5h10.5',
      'M6.25 4.5V3a.75.75 0 0 1 .75-.75h2a.75.75 0 0 1 .75.75v1.5',
      'M4 4.5l.6 8a1 1 0 0 0 1 .9h4.8a1 1 0 0 0 1-.9l.6-8',
      'M6.75 7v4M9.25 7v4',
    ],
  },
  // An undo arrow reads as "put it back".
  restore: { d: ['M5.5 3.5 2.75 6.25 5.5 9', 'M3 6.25h5a3.75 3.75 0 0 1 0 7.5H5.25'] },
  close: { d: ['M3.75 3.75l8.5 8.5', 'M12.25 3.75l-8.5 8.5'] },
  search: { d: ['M7 2.75a4.25 4.25 0 1 0 0 8.5 4.25 4.25 0 0 0 0-8.5z', 'M10.25 10.25l3 3'] },
  'chevron-left': { d: ['M10 3.25 5.25 8 10 12.75'] },
  'chevron-right': { d: ['M6 3.25 10.75 8 6 12.75'] },
  'chevron-down': { d: ['M3.5 6 8 10.5 12.5 6'] },
  external: {
    d: ['M9 2.75h4.25V7', 'M13.25 2.75 7.5 8.5', 'M11.25 9v3.25a1 1 0 0 1-1 1H3.75a1 1 0 0 1-1-1V5.75a1 1 0 0 1 1-1H7'],
  },
  warning: { d: ['M8 2.5 14 13H2z', 'M8 6.5v3'], dots: [[8, 11.4, 0.8]] },
  pin: {
    d: ['M8 14s4.5-4.1 4.5-7.5a4.5 4.5 0 0 0-9 0C3.5 9.9 8 14 8 14z', 'M8 5.25a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3z'],
  },
  key: { d: ['M5.25 8.25a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5z', 'M7 9.5l6.25-6.25', 'M11 5.5l1.75 1.75'] },
  gear: {
    d: [
      'M6.25 3.86L6.76 1.87H9.24L9.75 3.86L11.45 2.79L13.21 4.55L12.14 6.25L14.13 6.76V9.24L12.14 9.75L13.21 11.45L11.45 13.21L9.75 12.14L9.24 14.13H6.76L6.25 12.14L4.55 13.21L2.79 11.45L3.86 9.75L1.87 9.24V6.76L3.86 6.25L2.79 4.55L4.55 2.79z',
      'M8 6a2 2 0 1 0 0 4 2 2 0 0 0 0-4z',
    ],
  },
  pulse: { d: ['M1.75 8h3l1.75-4.5 3 9L11.25 8h3'] },
  image: {
    d: [rect(2.75, 2.75, 10.5, 10.5, 1.25), 'M2.75 11.25 6 8l2.5 2.5 1.75-1.75 3 3'],
    dots: [[10.5, 5.6, 0.85]],
  },
  more: { d: [], dots: [[3.5, 8, 1], [8, 8, 1], [12.5, 8, 1]] },
  plus: { d: ['M8 3.25v9.5', 'M3.25 8h9.5'] },
  // A strip of film: frame with sprocket ticks above and below.
  film: {
    d: [rect(1.75, 3.25, 12.5, 9.5, 1), 'M1.75 5.75h12.5', 'M1.75 10.25h12.5', 'M4.5 3.25v2.5M8 3.25v2.5M11.5 3.25v2.5', 'M4.5 10.25v2.5M8 10.25v2.5M11.5 10.25v2.5'],
  },
  // The darkroom safelight: a hanging lamp (theme switch).
  lamp: { d: ['M8 1.75v2.5', 'M4 9.25a4 4 0 0 1 8 0z', 'M6.25 11.75h3.5'], dots: [[8, 13.6, 0.8]] },
  loupe: { d: ['M6.75 2.75a4 4 0 1 0 0 8 4 4 0 0 0 0-8z', 'M9.75 9.75l3.5 3.5', 'M5 6.75h3.5M6.75 5v3.5'] },
  logout: {
    d: ['M6.25 2.75H4A1.25 1.25 0 0 0 2.75 4v8A1.25 1.25 0 0 0 4 13.25h2.25', 'M6.5 8h7', 'M10.75 5.25 13.5 8l-2.75 2.75'],
  },
} satisfies Record<string, IconShape>;

export type IconName = keyof typeof ICONS;
