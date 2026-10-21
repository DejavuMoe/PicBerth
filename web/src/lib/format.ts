// Display formatting. Sizes are 1024-based; times use Intl only (no date library).

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const;

/** 183 KB, 4.2 MB, 700.1 MB. Bytes and KB are integers, larger units carry one decimal. */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '—';
  if (bytes < 1024) return `${Math.round(bytes)} B`;
  let value = bytes / 1024;
  let unit = 1;
  // Promote when rounding would print 1024 (1023.6 KB -> 1.0 MB).
  const shown = () => (unit === 1 ? Math.round(value) : Math.round(value * 10) / 10);
  while (unit < UNITS.length - 1 && shown() >= 1024) {
    value /= 1024;
    unit++;
  }
  const name = UNITS[unit] ?? 'KB';
  return unit === 1 ? `${Math.round(value)} ${name}` : `${value.toFixed(1)} ${name}`;
}

/** Limits read better without a trailing .0 ("20 MB"). */
export function formatLimit(bytes: number): string {
  return formatBytes(bytes).replace('.0 ', ' ');
}

export function formatInt(n: number): string {
  return n.toLocaleString('en-US');
}

const pad = (n: number) => String(n).padStart(2, '0');

/** YYYY-MM-DD HH:mm in local time. */
export function formatAbsolute(iso: string | number | Date): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function formatDate(iso: string | number | Date): string {
  return formatAbsolute(iso).slice(0, 10);
}

const rtf0 = new Intl.RelativeTimeFormat('zh-CN', { numeric: 'auto' });
// Intl prints 3分钟前; the design wants a thin gap between the figure and the unit.
const rtf = {
  format: (v: number, unit: Intl.RelativeTimeFormatUnit) => rtf0.format(v, unit).replace(/(\d)(?=[\u4e00-\u9fff])/g, '$1 '),
};
const DAY = 86_400_000;

/** "3 分钟前", "昨天"; beyond 7 days falls back to the absolute time. */
export function formatRelative(iso: string | number | Date, now = Date.now()): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return '—';
  const diff = t - now;
  const abs = Math.abs(diff);
  if (abs >= 7 * DAY) return formatAbsolute(t);
  if (abs < 45_000) return '刚刚';
  if (abs < 3_600_000) return rtf.format(Math.round(diff / 60_000), 'minute');
  if (abs < DAY) return rtf.format(Math.round(diff / 3_600_000), 'hour');
  // Calendar-day difference so that "昨天" means yesterday, not "24 h ago".
  const a = new Date(t);
  const b = new Date(now);
  const days = Math.round(
    (Date.UTC(a.getFullYear(), a.getMonth(), a.getDate()) -
      Date.UTC(b.getFullYear(), b.getMonth(), b.getDate())) /
      DAY,
  );
  return rtf.format(days, 'day');
}

export function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d} 天 ${h} 小时`;
  if (h > 0) return `${h} 小时 ${m} 分`;
  if (m > 0) return `${m} 分 ${s % 60} 秒`;
  return `${s} 秒`;
}

export function formatDims(width: number, height: number): string {
  return `${width}×${height}`;
}

const MIME_LABEL: Record<string, string> = {
  'image/jpeg': 'JPEG',
  'image/png': 'PNG',
  'image/gif': 'GIF',
  'image/webp': 'WebP',
  'image/avif': 'AVIF',
};

export function formatLabel(mime: string, ext = ''): string {
  return MIME_LABEL[mime] ?? (ext ? ext.toUpperCase() : mime);
}

export const FORMAT_NAMES: Record<string, string> = {
  jpeg: 'JPEG',
  png: 'PNG',
  gif: 'GIF',
  webp: 'WebP',
  avif: 'AVIF',
};

export function formatNames(formats: readonly string[]): string[] {
  return formats.map((f) => FORMAT_NAMES[f] ?? f.toUpperCase());
}
