// Sanitized fixture data and pure helpers shared by all directions.

const NOW = new Date('2026-10-21T15:30:00+08:00').getTime();
const DAY = 86400000;
const IMG_HOST = 'https://img.example.com';

const PICS = {
  'dashboard-light.png': [1920, 1200, 'png'],
  'dashboard-dark.png': [1920, 1200, 'png'],
  'settings-panel.png': [1440, 960, 'png'],
  'mobile-login.png': [1170, 2340, 'png'],
  'code-snippet.png': [1600, 1000, 'png'],
  'bar-chart.png': [1200, 750, 'png'],
  'line-chart.png': [1600, 900, 'png'],
  'architecture.png': [1680, 980, 'png'],
  'mountain-dusk.jpg': [4032, 2688, 'jpeg'],
  'lake-morning.jpg': [3024, 2016, 'jpeg'],
  'desert-noon.jpg': [2268, 3402, 'jpeg'],
  'city-night.jpg': [3840, 2160, 'jpeg'],
  'receipt-scan.jpg': [1240, 1984, 'jpeg'],
  'logo-mark.png': [512, 512, 'png'],
  'avatar.png': [400, 400, 'png'],
  'banner-wide.webp': [1800, 600, 'webp'],
};

const TOKENS = [
  { id: 1, name: 'PicGo · 笔记本', prefix: 'pb_7KQ2M', scopes: ['upload'], createdAt: NOW - 62 * DAY, lastUsedAt: NOW - 2 * 3600000, expiresAt: null, revokedAt: null },
  { id: 2, name: 'ShareX · 工作机', prefix: 'pb_X4D9R', scopes: ['upload'], createdAt: NOW - 30 * DAY, lastUsedAt: NOW - 1 * DAY, expiresAt: NOW + 60 * DAY, revokedAt: null },
  { id: 3, name: '备份脚本', prefix: 'pb_A0N3T', scopes: ['read'], createdAt: NOW - 120 * DAY, lastUsedAt: NOW - 9 * DAY, expiresAt: null, revokedAt: null },
  { id: 4, name: '旧手机', prefix: 'pb_Q8W1Z', scopes: ['upload', 'manage'], createdAt: NOW - 200 * DAY, lastUsedAt: NOW - 90 * DAY, expiresAt: null, revokedAt: NOW - 40 * DAY },
];

const SOURCES = [
  { key: 'web', label: '网页' },
  { key: 'token:1', label: 'PicGo · 笔记本' },
  { key: 'token:2', label: 'ShareX · 工作机' },
  { key: 'cli', label: '命令行' },
];

// [file, displayName, minutesAgo, source, visibility, tags, gps, frames]
const ROWS = [
  ['dashboard-light.png', 'screenshot-2026-10-21-152210.png', 8, 'token:2', 'public', ['截图'], false, 1],
  ['code-snippet.png', 'handler-retry.png', 24, 'token:1', 'public', ['截图', '博客'], false, 1],
  ['architecture.png', '上传流程-v3.png', 70, 'web', 'public', ['文档', '架构'], false, 1],
  ['mountain-dusk.jpg', 'IMG_4821.jpg', 150, 'web', 'private', ['旅行'], true, 1],
  ['bar-chart.png', 'weekly-traffic.png', 300, 'token:1', 'public', ['博客'], false, 1],
  ['banner-wide.webp', 'post-cover-dark.webp', 1440 + 40, 'web', 'public', ['博客', '封面'], false, 1],
  ['dashboard-dark.png', 'screenshot-2026-10-20-221503.png', 1440 + 300, 'token:2', 'public', ['截图'], false, 1],
  ['receipt-scan.jpg', 'receipt-2026-10.jpg', 1440 + 600, 'web', 'private', ['票据'], false, 1],
  ['line-chart.png', 'latency-p95.png', 2 * 1440 + 100, 'token:1', 'public', ['博客'], false, 1],
  ['lake-morning.jpg', 'IMG_4790.jpg', 3 * 1440, 'web', 'private', ['旅行'], true, 1],
  ['logo-mark.png', 'logo-mark.png', 4 * 1440, 'cli', 'public', ['品牌'], false, 1],
  ['settings-panel.png', 'settings-before.png', 5 * 1440, 'token:2', 'public', ['截图', '文档'], false, 1],
  ['mobile-login.png', 'mobile-login.png', 6 * 1440, 'token:2', 'public', ['截图'], false, 1],
  ['avatar.png', 'avatar-2026.png', 9 * 1440, 'web', 'public', ['品牌'], false, 1],
  ['city-night.jpg', 'DSC_0112.jpg', 12 * 1440, 'web', 'public', ['旅行'], true, 1],
  ['desert-noon.jpg', 'DSC_0098.jpg', 12 * 1440 + 20, 'web', 'private', ['旅行'], true, 1],
  ['bar-chart.png', 'monthly-uploads.png', 18 * 1440, 'cli', 'public', ['文档'], false, 1],
  ['code-snippet.png', 'loading-spinner.gif', 22 * 1440, 'token:1', 'public', ['博客'], false, 24],
  ['architecture.png', '数据目录结构.png', 27 * 1440, 'web', 'public', ['文档', '架构'], false, 1],
  ['dashboard-light.png', 'screenshot-2026-09-22-101144.png', 29 * 1440, 'token:2', 'public', ['截图'], false, 1],
  ['banner-wide.webp', 'post-cover-light.webp', 35 * 1440, 'web', 'public', ['博客', '封面'], false, 1],
  ['lake-morning.jpg', 'IMG_4402.jpg', 41 * 1440, 'web', 'public', ['旅行'], false, 1],
  ['line-chart.png', 'error-rate.png', 48 * 1440, 'token:1', 'public', ['博客'], false, 1],
  ['receipt-scan.jpg', 'invoice-0912.jpg', 52 * 1440, 'web', 'private', ['票据'], false, 1],
  ['settings-panel.png', 'settings-after.png', 60 * 1440, 'token:2', 'public', ['截图', '文档'], false, 1],
  ['avatar.png', 'avatar-2025.png', 95 * 1440, 'web', 'public', ['品牌'], false, 1],
];

const TRASH_ROWS = [
  ['dashboard-dark.png', 'screenshot-old-draft.png', 3 * 1440, 'token:2', 'public', ['截图'], false, 1, 2],
  ['city-night.jpg', 'blurry-night.jpg', 20 * 1440, 'web', 'private', ['旅行'], true, 1, 11],
  ['code-snippet.png', 'typo-version.png', 7 * 1440, 'token:1', 'public', ['博客'], false, 1, 26],
];

const B32 = '0123456789abcdefghjkmnpqrstvwxyz';
function fakeId(seed) {
  let s = Math.imul(seed + 97, 2654435761) >>> 0 || 1;
  let out = '';
  for (let i = 0; i < 20; i++) {
    s ^= s << 13;
    s ^= s >>> 17;
    s ^= s << 5;
    s >>>= 0;
    out += B32[s % 32];
  }
  return out;
}

const MIME = { png: 'image/png', jpeg: 'image/jpeg', gif: 'image/gif', webp: 'image/webp', avif: 'image/avif' };
const FORMAT_LABEL = { jpeg: 'JPEG', png: 'PNG', gif: 'GIF', webp: 'WebP', avif: 'AVIF' };

function makeImage(row, i, trashed) {
  const [file, name, minutesAgo, source, visibility, tags, gps, frames, trashedDaysAgo] = row;
  const [w, h, fmt0] = PICS[file];
  const fmt = frames > 1 ? 'gif' : fmt0;
  const ext = fmt === 'jpeg' ? 'jpg' : fmt;
  const id = fakeId(i + (trashed ? 500 : 1));
  const url = `${IMG_HOST}/i/${id}.${ext}`;
  const size = Math.round((w * h * (fmt === 'png' ? 0.55 : fmt === 'webp' ? 0.12 : 0.22) * (0.7 + ((i * 37) % 60) / 100)) / 1);
  return {
    id,
    filename: name,
    thumb: `assets/fixtures/${file}`,
    format: fmt,
    mime: MIME[fmt],
    ext,
    width: w,
    height: h,
    size,
    frames,
    visibility,
    hasGps: gps,
    source,
    tags: [...tags],
    sha256: (id + id + id + id).slice(0, 64).replace(/[^0-9a-f]/g, (c) => B32.indexOf(c).toString(16).slice(-1)),
    createdAt: NOW - minutesAgo * 60000,
    trashedAt: trashed ? NOW - trashedDaysAgo * DAY : null,
    state: trashed ? 'trashed' : 'active',
    url,
    links: {
      url,
      markdown: `![${name}](${url})`,
      html: `<img src="${url}" alt="${name}">`,
      bbcode: `[img]${url}[/img]`,
    },
  };
}

function initialImages() {
  return [...ROWS.map((r, i) => makeImage(r, i, false)), ...TRASH_ROWS.map((r, i) => makeImage(r, i, true))];
}

const SETTINGS = {
  maxUploadMB: 20,
  hardMaxMB: 100,
  maxPixelsM: 50,
  allowedFormats: ['jpeg', 'png', 'gif', 'webp', 'avif'],
  defaultVisibility: 'public',
  dedupe: true,
  trashRetentionDays: 30,
  uploadsPerMinute: 60,
  sessionIdleMinutes: 30,
  sessionAbsoluteHours: 12,
  publicUrl: 'https://pics.example.com',
  imageUrl: 'https://img.example.com',
};

const STATUS = {
  version: '0.2.0',
  commit: '4f1c2ab',
  goVersion: 'go1.27.3',
  sqliteVersion: '3.53.4',
  uptimeSeconds: 6 * 86400 + 4 * 3600,
  dataDir: '/var/lib/picberth',
  db: { bytes: 1.4 * 1048576, walBytes: 220 * 1024 },
  disk: { freeBytes: 182 * 1073741824, totalBytes: 250 * 1073741824, watermarkBytes: 5 * 1073741824 },
  lastSnapshot: { at: NOW - 9 * DAY, path: '/var/lib/picberth/snapshots/20261012T030000Z' },
  lastCheck: { at: NOW - 20 * 3600000, mode: 'quick', ok: true, issues: 0 },
  warnings: [{ code: 'snapshot-stale', title: '超过 7 天没有创建快照', detail: '在下方创建一次快照，再用备份工具保存快照目录。' }],
};

const AUDIT = [
  { at: NOW - 8 * 60000, actor: 'ShareX · 工作机', action: '上传', target: 'screenshot-2026-10-21-152210.png' },
  { at: NOW - 24 * 60000, actor: 'PicGo · 笔记本', action: '上传', target: 'handler-retry.png' },
  { at: NOW - 3 * 3600000, actor: 'admin', action: '设为私有', target: 'IMG_4821.jpg' },
  { at: NOW - 1 * DAY, actor: 'admin', action: '移到回收站', target: 'screenshot-old-draft.png' },
  { at: NOW - 2 * DAY, actor: 'admin', action: '修改设置', target: '回收站保留天数' },
  { at: NOW - 9 * DAY, actor: 'admin', action: '创建快照', target: '20261012T030000Z' },
];

// ---------------------------------------------------------------- formatting

function fmtBytes(n) {
  if (n < 1024) return `${n} B`;
  const u = ['KB', 'MB', 'GB', 'TB'];
  let v = n / 1024;
  let i = 0;
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024;
    i++;
  }
  return i === 0 ? `${Math.round(v)} ${u[i]}` : `${v.toFixed(1)} ${u[i]}`;
}
const pad2 = (n) => String(n).padStart(2, '0');
function fmtDate(t) {
  const d = new Date(t);
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}
function fmtRel(t) {
  const diff = NOW - t;
  if (diff < 0) {
    const days = Math.round(-diff / DAY);
    return `${days} 天后`;
  }
  if (diff < 60000) return '刚刚';
  if (diff < 3600000) return `${Math.round(diff / 60000)} 分钟前`;
  if (diff < DAY) return `${Math.round(diff / 3600000)} 小时前`;
  const days = Math.floor((startOfDay(NOW) - startOfDay(t)) / DAY);
  if (days <= 1) return '昨天';
  if (days < 7) return `${days} 天前`;
  return fmtDate(t).slice(0, 10);
}
function startOfDay(t) {
  const d = new Date(t);
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}
const WEEK = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
function dayLabel(t) {
  const days = Math.round((startOfDay(NOW) - startOfDay(t)) / DAY);
  const d = new Date(t);
  const date = `${d.getMonth() + 1}月${d.getDate()}日`;
  if (days === 0) return ['今天', `${date} ${WEEK[d.getDay()]}`];
  if (days === 1) return ['昨天', `${date} ${WEEK[d.getDay()]}`];
  return [date, WEEK[d.getDay()]];
}
function sourceLabel(key) {
  return SOURCES.find((s) => s.key === key)?.label ?? key;
}
function daysLeft(img, retention) {
  const left = Math.ceil((img.trashedAt + retention * DAY - NOW) / DAY);
  return left <= 0 ? '即将清除' : `${left} 天后清除`;
}

// ---------------------------------------------------------------- filtering

const DATE_RANGES = [
  { key: 'today', label: '今天', since: () => startOfDay(NOW) },
  { key: '7d', label: '最近 7 天', since: () => NOW - 7 * DAY },
  { key: '30d', label: '最近 30 天', since: () => NOW - 30 * DAY },
  { key: 'older', label: '30 天以前', until: () => NOW - 30 * DAY },
];

const EMPTY_FILTERS = { q: '', formats: [], vis: '', date: '', sources: [], tags: [], gps: false };

function applyFilters(list, f, sort, ext) {
  const q = f.q.trim().toLowerCase();
  let out = list.filter((im) => {
    if (q && !(im.filename.toLowerCase().includes(q) || im.id.includes(q) || im.sha256.startsWith(q) || (ext && im.tags.some((t) => t.includes(q))))) return false;
    if (f.formats.length && !f.formats.includes(im.format)) return false;
    if (f.vis && im.visibility !== f.vis) return false;
    if (f.gps && !im.hasGps) return false;
    if (ext) {
      if (f.date) {
        const r = DATE_RANGES.find((d) => d.key === f.date);
        if (r?.since && im.createdAt < r.since()) return false;
        if (r?.until && im.createdAt >= r.until()) return false;
      }
      if (f.sources.length && !f.sources.includes(im.source)) return false;
      if (f.tags.length && !f.tags.every((t) => im.tags.includes(t))) return false;
    }
    return true;
  });
  const by = {
    new: (a, b) => b.createdAt - a.createdAt,
    old: (a, b) => a.createdAt - b.createdAt,
    large: (a, b) => b.size - a.size,
    small: (a, b) => a.size - b.size,
    name: (a, b) => a.filename.localeCompare(b.filename),
  }[sort] ?? ((a, b) => b.createdAt - a.createdAt);
  return [...out].sort(by);
}

function activeFilterCount(f, ext) {
  return (f.q ? 1 : 0) + f.formats.length + (f.vis ? 1 : 0) + (f.gps ? 1 : 0) + (ext ? (f.date ? 1 : 0) + f.sources.length + f.tags.length : 0);
}

function allTags(list) {
  const m = new Map();
  for (const im of list) for (const t of im.tags) m.set(t, (m.get(t) ?? 0) + 1);
  return [...m.entries()].sort((a, b) => b[1] - a[1]);
}

function groupByDay(list) {
  const out = [];
  for (const im of list) {
    const key = startOfDay(im.createdAt);
    let g = out[out.length - 1];
    if (!g || g.key !== key) {
      g = { key, label: dayLabel(im.createdAt), items: [] };
      out.push(g);
    }
    g.items.push(im);
  }
  return out;
}

const LINK_FORMATS = [
  { key: 'url', label: 'URL' },
  { key: 'markdown', label: 'Markdown' },
  { key: 'html', label: 'HTML' },
  { key: 'bbcode', label: 'BBCode' },
];
const SORTS = [
  { key: 'new', label: '最新上传' },
  { key: 'old', label: '最早上传' },
  { key: 'large', label: '文件最大' },
  { key: 'small', label: '文件最小' },
  { key: 'name', label: '文件名' },
];

Object.assign(window, {
  NOW, DAY, TOKENS, SOURCES, SETTINGS, STATUS, AUDIT, FORMAT_LABEL, MIME, LINK_FORMATS, SORTS, DATE_RANGES,
  EMPTY_FILTERS, initialImages, fmtBytes, fmtDate, fmtRel, dayLabel, sourceLabel, daysLeft, applyFilters,
  activeFilterCount, allTags, groupByDay, fakeId,
});
