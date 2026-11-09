// Stroke icons on a 16 px grid (1.5 px stroke, round caps), from the approved prototype (ui.jsx PATHS).
// Shapes are drawn by components/Icon.svelte with styling from CSS (.icon, .icon-s).

export type IconShape =
  | { t: 'path'; d: string }
  | { t: 'circle'; cx: number; cy: number; r: number }
  | { t: 'rect'; x: number; y: number; w: number; h: number; rx?: number };

export const ICONS = {
  image: [
    { t: 'rect', x: 2.5, y: 2.5, w: 11, h: 11, rx: 2 },
    { t: 'circle', cx: 6, cy: 6.25, r: 1.1 },
    { t: 'path', d: 'M13.5 10.5 10.5 7.5 3.5 13.5' },
  ],
  trash: [
    { t: 'path', d: 'M3 4.5h10' },
    { t: 'path', d: 'M6.5 4.5V3h3v1.5' },
    { t: 'path', d: 'M4.5 4.5l.5 8.5h6l.5-8.5' },
  ],
  key: [
    { t: 'circle', cx: 5.5, cy: 10.5, r: 2.5 },
    { t: 'path', d: 'M7.3 8.7 13 3' },
    { t: 'path', d: 'M11 5l1.5 1.5' },
    { t: 'path', d: 'M9.5 6.5l1.2 1.2' },
  ],
  sliders: [
    { t: 'path', d: 'M3 5h6' },
    { t: 'path', d: 'M12 5h1' },
    { t: 'path', d: 'M3 11h1' },
    { t: 'path', d: 'M7 11h6' },
    { t: 'circle', cx: 10.5, cy: 5, r: 1.5 },
    { t: 'circle', cx: 5.5, cy: 11, r: 1.5 },
  ],
  pulse: [{ t: 'path', d: 'M2 8h2.5L6 4.5l3 7L10.5 8H14' }],
  search: [
    { t: 'circle', cx: 7, cy: 7, r: 4.25 },
    { t: 'path', d: 'm10.25 10.25 3.25 3.25' },
  ],
  filter: [
    { t: 'path', d: 'M2.5 4h11' },
    { t: 'path', d: 'M4.5 8h7' },
    { t: 'path', d: 'M6.5 12h3' },
  ],
  upload: [
    { t: 'path', d: 'M8 10.5V2.75' },
    { t: 'path', d: 'M4.75 6 8 2.75 11.25 6' },
    { t: 'path', d: 'M3 10v2.5c0 .55.45 1 1 1h8c.55 0 1-.45 1-1V10' },
  ],
  copy: [
    { t: 'rect', x: 5.5, y: 5.5, w: 8, h: 8, rx: 1.5 },
    { t: 'path', d: 'M10.5 5.5V4a1.5 1.5 0 0 0-1.5-1.5H4A1.5 1.5 0 0 0 2.5 4v5A1.5 1.5 0 0 0 4 10.5h1.5' },
  ],
  check: [{ t: 'path', d: 'm3.5 8.5 3 3 6-7' }],
  minus: [{ t: 'path', d: 'M4 8h8' }],
  x: [
    { t: 'path', d: 'm4 4 8 8' },
    { t: 'path', d: 'M12 4l-8 8' },
  ],
  more: [
    { t: 'circle', cx: 3.5, cy: 8, r: 0.6 },
    { t: 'circle', cx: 8, cy: 8, r: 0.6 },
    { t: 'circle', cx: 12.5, cy: 8, r: 0.6 },
  ],
  left: [{ t: 'path', d: 'M10 3.5 5.5 8l4.5 4.5' }],
  right: [{ t: 'path', d: 'M6 3.5 10.5 8 6 12.5' }],
  down: [{ t: 'path', d: 'M4 6l4 4 4-4' }],
  list: [
    { t: 'path', d: 'M5.5 4h8' },
    { t: 'path', d: 'M5.5 8h8' },
    { t: 'path', d: 'M5.5 12h8' },
    { t: 'path', d: 'M2.5 4h.01' },
    { t: 'path', d: 'M2.5 8h.01' },
    { t: 'path', d: 'M2.5 12h.01' },
  ],
  grid: [
    { t: 'rect', x: 2.5, y: 2.5, w: 4.5, h: 4.5, rx: 1 },
    { t: 'rect', x: 9, y: 2.5, w: 4.5, h: 4.5, rx: 1 },
    { t: 'rect', x: 2.5, y: 9, w: 4.5, h: 4.5, rx: 1 },
    { t: 'rect', x: 9, y: 9, w: 4.5, h: 4.5, rx: 1 },
  ],
  sort: [
    { t: 'path', d: 'M5 3v10' },
    { t: 'path', d: 'M2.5 10.5 5 13l2.5-2.5' },
    { t: 'path', d: 'M11 13V3' },
    { t: 'path', d: 'M8.5 5.5 11 3l2.5 2.5' },
  ],
  download: [
    { t: 'path', d: 'M8 2.75v7.75' },
    { t: 'path', d: 'M4.75 7.25 8 10.5l3.25-3.25' },
    { t: 'path', d: 'M3 13.25h10' },
  ],
  restore: [
    { t: 'path', d: 'M3 8a5 5 0 1 0 1.5-3.55' },
    { t: 'path', d: 'M3 2.75V5h2.25' },
  ],
  plus: [
    { t: 'path', d: 'M8 3v10' },
    { t: 'path', d: 'M3 8h10' },
  ],
  select: [
    { t: 'rect', x: 2.5, y: 2.5, w: 11, h: 11, rx: 2 },
    { t: 'path', d: 'm5.5 8 1.75 1.75L10.75 6' },
  ],
  sun: [
    { t: 'circle', cx: 8, cy: 8, r: 2.75 },
    { t: 'path', d: 'M8 1.75v1.5' },
    { t: 'path', d: 'M8 12.75v1.5' },
    { t: 'path', d: 'M1.75 8h1.5' },
    { t: 'path', d: 'M12.75 8h1.5' },
    { t: 'path', d: 'M3.6 3.6l1 1' },
    { t: 'path', d: 'M11.4 11.4l1 1' },
    { t: 'path', d: 'M3.6 12.4l1-1' },
    { t: 'path', d: 'M11.4 4.6l1-1' },
  ],
  logout: [
    { t: 'path', d: 'M9.5 3H4a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h5.5' },
    { t: 'path', d: 'M7 8h6.5' },
    { t: 'path', d: 'M11 5.5 13.5 8 11 10.5' },
  ],
  alert: [
    { t: 'circle', cx: 8, cy: 8, r: 5.5 },
    { t: 'path', d: 'M8 5v3.25' },
    { t: 'path', d: 'M8 10.75h.01' },
  ],
  refresh: [
    { t: 'path', d: 'M13 8a5 5 0 1 1-1.5-3.55' },
    { t: 'path', d: 'M13 2.75V5h-2.25' },
  ],
  edit: [{ t: 'path', d: 'M10.5 3 13 5.5 6 12.5H3.5V10Z' }],
  folder: [{ t: 'path', d: 'M2.5 4.5a1 1 0 0 1 1-1h2.9l1.2 1.5h4.9a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1Z' }],
  'folder-plus': [
    { t: 'path', d: 'M8.5 12.5h-5a1 1 0 0 1-1-1v-7a1 1 0 0 1 1-1h2.9l1.2 1.5h4.9a1 1 0 0 1 1 1V8' },
    { t: 'path', d: 'M12 10v4' },
    { t: 'path', d: 'M10 12h4' },
  ],
  external: [
    { t: 'path', d: 'M9 3h4v4' },
    { t: 'path', d: 'M13 3 7.5 8.5' },
    { t: 'path', d: 'M11 9.5V12a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h2.5' },
  ],
} satisfies Record<string, IconShape[]>;

export type IconName = keyof typeof ICONS;
