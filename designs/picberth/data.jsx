// Sanitized fixtures and pure helpers for the prototype.

const NOW = new Date('2026-11-08T15:30:00+08:00').getTime();
const MIN = 60000;
const HOUR = 60 * MIN;
const DAY = 24 * HOUR;
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
  { id: 1, name: 'PicGo 笔记本', prefix: 'pb_7KQ2M', scopes: ['upload'], createdAt: NOW - 62 * DAY, lastUsedAt: NOW - 2 * HOUR, expiresAt: null, revokedAt: null },
  { id: 2, name: 'ShareX 工作机', prefix: 'pb_X4D9R', scopes: ['upload'], createdAt: NOW - 30 * DAY, lastUsedAt: NOW - 8 * MIN, expiresAt: NOW + 60 * DAY, revokedAt: null },
  { id: 3, name: '备份脚本', prefix: 'pb_A0N3T', scopes: ['read'], createdAt: NOW - 120 * DAY, lastUsedAt: NOW - 9 * DAY, expiresAt: null, revokedAt: null },
  { id: 5, name: '博客构建', prefix: 'pb_M2V6C', scopes: ['upload', 'read'], createdAt: NOW - 10 * DAY, lastUsedAt: null, expiresAt: NOW + 20 * DAY, revokedAt: null },
  { id: 4, name: '旧手机', prefix: 'pb_Q8W1Z', scopes: ['upload', 'manage'], createdAt: NOW - 200 * DAY, lastUsedAt: NOW - 90 * DAY, expiresAt: null, revokedAt: NOW - 40 * DAY },
];

const SOURCES = [
  { key: 'web', label: '网页' },
  { key: 'token:1', label: 'PicGo 笔记本' },
  { key: 'token:2', label: 'ShareX 工作机' },
  { key: 'cli', label: '命令行' },
];

// [fixture, filename, minutesAgo, source, tags, frames]
const ROWS = [
  ['dashboard-light.png', 'screenshot-2026-11-08-152210.png', 8, 'token:2', ['截图'], 1],
  ['code-snippet.png', 'handler-retry.png', 24, 'token:1', ['截图', '博客'], 1],
  ['architecture.png', '上传流程-v3.png', 70, 'web', ['文档', '架构'], 1],
  ['mountain-dusk.jpg', 'IMG_4821.jpg', 150, 'web', ['旅行'], 1],
  ['bar-chart.png', 'weekly-traffic.png', 300, 'token:1', ['博客'], 1],
  ['banner-wide.webp', 'post-cover-dark.webp', 1440 + 40, 'web', ['博客', '封面'], 1],
  ['dashboard-dark.png', 'screenshot-2026-11-07-221503.png', 1440 + 300, 'token:2', ['截图'], 1],
  ['receipt-scan.jpg', 'receipt-2026-11.jpg', 1440 + 600, 'web', ['票据'], 1],
  ['line-chart.png', 'latency-p95.png', 2 * 1440 + 100, 'token:1', ['博客'], 1],
  ['lake-morning.jpg', 'IMG_4790.jpg', 3 * 1440, 'web', ['旅行'], 1],
  ['logo-mark.png', 'logo-mark.png', 4 * 1440, 'cli', ['品牌'], 1],
  ['settings-panel.png', 'settings-before.png', 5 * 1440, 'token:2', ['截图', '文档'], 1],
  ['mobile-login.png', 'mobile-login.png', 6 * 1440, 'token:2', ['截图'], 1],
  ['avatar.png', 'avatar-2026.png', 9 * 1440, 'web', ['品牌'], 1],
  ['city-night.jpg', 'DSC_0112.jpg', 12 * 1440, 'web', ['旅行'], 1],
  ['desert-noon.jpg', 'DSC_0098.jpg', 12 * 1440 + 20, 'web', ['旅行'], 1],
  ['bar-chart.png', 'monthly-uploads.png', 18 * 1440, 'cli', ['文档'], 1],
  ['code-snippet.png', 'loading-spinner.gif', 22 * 1440, 'token:1', ['博客'], 24],
  ['architecture.png', '数据目录结构.png', 27 * 1440, 'web', ['文档', '架构'], 1],
  ['dashboard-light.png', 'screenshot-2026-10-10-101144.png', 29 * 1440, 'token:2', ['截图'], 1],
  ['banner-wide.webp', 'post-cover-light.webp', 35 * 1440, 'web', ['博客', '封面'], 1],
  ['lake-morning.jpg', 'IMG_4402.jpg', 41 * 1440, 'web', ['旅行'], 1],
  ['line-chart.png', 'error-rate.png', 48 * 1440, 'token:1', ['博客'], 1],
  ['receipt-scan.jpg', 'invoice-0912.jpg', 52 * 1440, 'web', ['票据'], 1],
];

// [..., trashedDaysAgo]
const TRASH_ROWS = [
  ['dashboard-dark.png', 'screenshot-old-draft.png', 3 * 1440, 'token:2', ['截图'], 1, 2],
  ['code-snippet.png', 'typo-version.png', 7 * 1440, 'token:1', ['博客'], 1, 26],
  ['city-night.jpg', 'blurry-night.jpg', 20 * 1440, 'web', ['旅行'], 1, 11],
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

const FORMAT_LABEL = { jpeg: 'JPEG', png: 'PNG', gif: 'GIF', webp: 'WebP', avif: 'AVIF' };

function makeLinks(id, ext, name) {
  const url = `${IMG_HOST}/i/${id}.${ext}`;
  return {
    url,
    markdown: `![${name}](${url})`,
    html: `<img src="${url}" alt="${name}">`,
    bbcode: `[img]${url}[/img]`,
  };
}

function makeImage(row, i, trashed) {
  const [file, name, minutesAgo, source, tags, frames, trashedDaysAgo] = row;
  const [w, h, fmt0] = PICS[file];
  const format = frames > 1 ? 'gif' : fmt0;
  const ext = format === 'jpeg' ? 'jpg' : format;
  const id = fakeId(i + (trashed ? 500 : 1));
  const size = Math.round(w * h * (format === 'png' ? 0.55 : format === 'webp' ? 0.12 : 0.22) * (0.7 + ((i * 37) % 60) / 100));
  const sha = Array.from({ length: 64 }, (_, k) => '0123456789abcdef'[(id.charCodeAt(k % 20) * (k + 7)) % 16]).join('');
  return {
    id, filename: name, thumb: `assets/fixtures/${file}`, format, ext, width: w, height: h, size, frames,
    source, tags: [...tags], sha256: sha, createdAt: NOW - minutesAgo * MIN,
    trashedAt: trashed ? NOW - trashedDaysAgo * DAY : null, state: trashed ? 'trashed' : 'active',
    links: makeLinks(id, ext, name),
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
  uploadsPerMinute: 60,
  dedupe: true,
  trashRetentionDays: 30,
  sessionIdleMinutes: 30,
  sessionAbsoluteHours: 12,
  publicUrl: 'https://pics.example.com',
  imageUrl: 'https://img.example.com',
};

const STATUS = {
  version: '0.3.0',
  commit: '4f1c2ab',
  goVersion: 'go1.27.3',
  sqliteVersion: '3.53.4',
  uptime: '6 天',
  dataDir: '/var/lib/picberth',
  images: { active: 1284, trashed: 3, bytes: 2.31 * 1073741824, broken: 0 },
  dbBytes: 1.4 * 1048576,
  disk: { free: 182 * 1073741824, total: 250 * 1073741824 },
  lastSnapshot: NOW - 9 * DAY,
  lastQuick: { at: NOW - 20 * HOUR, ok: true },
  lastFull: { at: NOW - 16 * DAY, ok: true, checked: 1287 },
  warnings: [
    { code: 'snapshot-stale', title: '近期没有快照', help: '上次快照在 9 天前' },
  ],
};

const AUDIT = [
  { at: NOW - 8 * MIN, actor: 'ShareX 工作机（#2）', action: '上传', target: 'screenshot-2026-11-08-152210.png' },
  { at: NOW - 24 * MIN, actor: 'PicGo 笔记本（#1）', action: '上传', target: 'handler-retry.png' },
  { at: NOW - 3 * HOUR, actor: 'admin', action: '重命名', target: 'IMG_4821.jpg' },
  { at: NOW - 20 * HOUR, actor: '系统', action: '快速检查', target: '1287 张图片' },
  { at: NOW - 1 * DAY, actor: 'admin', action: '移到回收站', target: 'screenshot-old-draft.png' },
  { at: NOW - 2 * DAY, actor: 'admin', action: '修改设置', target: '回收站保留时间' },
  { at: NOW - 9 * DAY, actor: 'admin', action: '创建快照', target: '20261030T030000Z' },
  { at: NOW - 10 * DAY, actor: 'admin', action: '创建令牌', target: '博客构建' },
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
function startOfDay(t) {
  const d = new Date(t);
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}
function fmtRel(t) {
  const diff = NOW - t;
  if (diff < 0) return `${Math.round(-diff / DAY)} 天后`;
  if (diff < MIN) return '刚刚';
  if (diff < HOUR) return `${Math.round(diff / MIN)} 分钟前`;
  if (diff < DAY) return `${Math.round(diff / HOUR)} 小时前`;
  const days = Math.round((startOfDay(NOW) - startOfDay(t)) / DAY);
  if (days <= 1) return '昨天';
  if (days < 7) return `${days} 天前`;
  return fmtDate(t).slice(0, 10);
}
function sourceLabel(key) {
  return SOURCES.find((s) => s.key === key)?.label ?? key;
}
function purgeIn(img, retention) {
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
const EMPTY_FILTERS = { q: '', formats: [], date: '', sources: [], tags: [] };

function applyFilters(list, f, sort) {
  const q = f.q.trim().toLowerCase();
  const out = list.filter((im) => {
    if (q && !(im.filename.toLowerCase().includes(q) || im.id.startsWith(q) || im.sha256.startsWith(q) || im.tags.some((t) => t.includes(q)))) return false;
    if (f.formats.length && !f.formats.includes(im.format)) return false;
    if (f.date) {
      const r = DATE_RANGES.find((d) => d.key === f.date);
      if (r?.since && im.createdAt < r.since()) return false;
      if (r?.until && im.createdAt >= r.until()) return false;
    }
    if (f.sources.length && !f.sources.includes(im.source)) return false;
    if (f.tags.length && !f.tags.every((t) => im.tags.includes(t))) return false;
    return true;
  });
  const by = {
    new: (a, b) => b.createdAt - a.createdAt,
    old: (a, b) => a.createdAt - b.createdAt,
    large: (a, b) => b.size - a.size,
    small: (a, b) => a.size - b.size,
    name: (a, b) => a.filename.localeCompare(b.filename),
  }[sort];
  return [...out].sort(by);
}

function filterCount(f) {
  return f.formats.length + (f.date ? 1 : 0) + f.sources.length + f.tags.length;
}

function tagCounts(list) {
  const m = new Map();
  for (const im of list) for (const t of im.tags) m.set(t, (m.get(t) ?? 0) + 1);
  return [...m.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
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
const SCOPES = [
  { key: 'upload', label: '上传' },
  { key: 'read', label: '只读' },
  { key: 'manage', label: '管理' },
];

Object.assign(window, {
  NOW, MIN, HOUR, DAY, IMG_HOST, TOKENS, SOURCES, SETTINGS, STATUS, AUDIT, FORMAT_LABEL, LINK_FORMATS, SORTS, SCOPES,
  DATE_RANGES, EMPTY_FILTERS, initialImages, makeLinks, fakeId, fmtBytes, fmtDate, fmtRel, sourceLabel, purgeIn,
  applyFilters, filterCount, tagCounts,
});
