// Display formatting. Sizes are 1024-based; times use plain arithmetic (no date library).

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

const DAY = 86_400_000;

export function startOfDay(t: number | Date): number {
  const d = new Date(t);
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

/** "8 分钟前", "昨天", "3 天前"; from 7 days on the date alone. */
export function formatRelative(iso: string | number | Date, now = Date.now()): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return '—';
  const diff = now - t;
  if (diff < 0) return `${Math.max(1, Math.round(-diff / DAY))} 天后`;
  if (diff < 60_000) return '刚刚';
  if (diff < 3_600_000) return `${Math.round(diff / 60_000)} 分钟前`;
  if (diff < DAY) return `${Math.round(diff / 3_600_000)} 小时前`;
  const days = Math.round((startOfDay(now) - startOfDay(t)) / DAY);
  if (days <= 1) return '昨天';
  if (days < 7) return `${days} 天前`;
  return formatDate(t);
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

// Approximate advance widths at 13 px: CJK and full-width ≈ 13, Latin ≈ 7.
const charWidth = (c: string) => (/[⺀-￿]/.test(c) ? 13 : 7);
const textWidth = (s: string) => [...s].reduce((a, c) => a + charWidth(c), 0);

/**
 * Shorten a file name in the middle so it fits about `px` wide, keeping the start and the extension
 * (screenshot-2026…52210.png), so similar names stay distinguishable.
 */
export function middleEllipsis(name: string, px: number): string {
  if (textWidth(name) <= px) return name;
  const dot = name.lastIndexOf('.');
  const ext = dot > 0 && name.length - dot <= 6 ? name.slice(dot) : '';
  const chars = [...(ext ? name.slice(0, dot) : name)];
  let budget = px - textWidth(ext) - 13;
  let head = '';
  let tail = '';
  let i = 0;
  let j = chars.length - 1;
  while (i <= j && budget > 0) {
    if (textWidth(head) <= textWidth(tail) * 1.6) {
      const c = chars[i]!;
      budget -= charWidth(c);
      if (budget >= 0) {
        head += c;
        i++;
      }
    } else {
      const c = chars[j]!;
      budget -= charWidth(c);
      if (budget >= 0) {
        tail = c + tail;
        j--;
      }
    }
  }
  return `${head}…${tail}${ext}`;
}
