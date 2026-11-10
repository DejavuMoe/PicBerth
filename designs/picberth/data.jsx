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
  { id: 3, name: '备份脚本', prefix: 'pb_A0N3T', scopes: ['manage'], createdAt: NOW - 120 * DAY, lastUsedAt: NOW - 9 * DAY, expiresAt: null, revokedAt: null },
  { id: 5, name: '博客构建', prefix: 'pb_M2V6C', scopes: ['upload', 'manage'], createdAt: NOW - 10 * DAY, lastUsedAt: null, expiresAt: NOW + 20 * DAY, revokedAt: null },
  { id: 4, name: '旧手机', prefix: 'pb_Q8W1Z', scopes: ['upload', 'manage'], createdAt: NOW - 200 * DAY, lastUsedAt: NOW - 90 * DAY, expiresAt: null, revokedAt: NOW - 40 * DAY },
];

const SOURCES = [
  { key: 'web', label: '网页' },
  { key: 'token:1', label: 'PicGo 笔记本' },
  { key: 'token:2', label: 'ShareX 工作机' },
  { key: 'cli', label: '命令行' },
];

// Folders: depth at most 3 (root-level folder = depth 1). Image rows point at a folder id or null (root).
const MAX_DEPTH = 3;
const FOLDERS = [
  { id: 'travel', name: '旅行', parentId: null },
  { id: 'y2026', name: '2026', parentId: 'travel' },
  { id: 'yunnan', name: '云南', parentId: 'y2026' },
  { id: 'y2025', name: '2025', parentId: 'travel' },
  { id: 'blog', name: '博客', parentId: null },
  { id: 'covers', name: '封面', parentId: 'blog' },
  { id: 'shots', name: '截图', parentId: null },
  { id: 'docs', name: '文档', parentId: null },
  { id: 'architecture', name: '架构', parentId: 'docs' },
  { id: 'brand', name: '品牌', parentId: null },
  { id: 'receipts', name: '票据', parentId: null },
];

// [fixture, filename, minutesAgo, source, folderId, frames]
const ROWS = [
  ['dashboard-light.png', 'screenshot-2026-11-08-152210.png', 8, 'token:2', null, 1],
  ['code-snippet.png', 'handler-retry.png', 24, 'token:1', null, 1],
  ['architecture.png', '上传流程-v3.png', 70, 'web', 'architecture', 1],
  ['mountain-dusk.jpg', 'IMG_4821.jpg', 150, 'web', 'y2026', 1],
  ['bar-chart.png', 'weekly-traffic.png', 300, 'token:1', 'blog', 1],
  ['banner-wide.webp', 'post-cover-dark.webp', 1440 + 40, 'web', null, 1],
  ['dashboard-dark.png', 'screenshot-2026-11-07-221503.png', 1440 + 300, 'token:2', 'shots', 1],
  ['receipt-scan.jpg', 'receipt-2026-11.jpg', 1440 + 600, 'web', 'receipts', 1],
  ['line-chart.png', 'latency-p95.png', 2 * 1440 + 100, 'token:1', null, 1],
  ['lake-morning.jpg', 'IMG_4790.jpg', 3 * 1440, 'web', 'y2026', 1],
  ['logo-mark.png', 'logo-mark.png', 4 * 1440, 'cli', 'brand', 1],
  ['settings-panel.png', 'settings-before.png', 5 * 1440, 'token:2', 'shots', 1],
  ['mobile-login.png', 'mobile-login.png', 6 * 1440, 'token:2', null, 1],
  ['avatar.png', 'avatar-2026.png', 9 * 1440, 'web', 'brand', 1],
  ['city-night.jpg', 'DSC_0112.jpg', 12 * 1440, 'web', 'yunnan', 1],
  ['desert-noon.jpg', 'DSC_0098.jpg', 12 * 1440 + 20, 'web', null, 1],
  ['bar-chart.png', 'monthly-uploads.png', 18 * 1440, 'cli', null, 1],
  ['code-snippet.png', 'loading-spinner.gif', 22 * 1440, 'token:1', 'blog', 24],
  ['architecture.png', '数据目录结构.png', 27 * 1440, 'web', 'architecture', 1],
  ['dashboard-light.png', 'screenshot-2026-10-10-101144.png', 29 * 1440, 'token:2', 'shots', 1],
  ['banner-wide.webp', 'post-cover-light.webp', 35 * 1440, 'web', 'covers', 1],
  ['lake-morning.jpg', 'IMG_4402.jpg', 41 * 1440, 'web', null, 1],
  ['line-chart.png', 'error-rate.png', 48 * 1440, 'token:1', 'blog', 1],
  ['receipt-scan.jpg', 'invoice-0912.jpg', 52 * 1440, 'web', 'receipts', 1],
];

// [..., trashedDaysAgo]
const TRASH_ROWS = [
  ['dashboard-dark.png', 'screenshot-old-draft.png', 3 * 1440, 'token:2', 'shots', 1, 2],
  ['code-snippet.png', 'typo-version.png', 7 * 1440, 'token:1', 'blog', 1, 26],
  ['city-night.jpg', 'blurry-night.jpg', 20 * 1440, 'web', 'yunnan', 1, 11],
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
  const [file, name, minutesAgo, source, folderId, frames, trashedDaysAgo] = row;
  const [w, h, fmt0] = PICS[file];
  const format = frames > 1 ? 'gif' : fmt0;
  const ext = format === 'jpeg' ? 'jpg' : format;
  const id = fakeId(i + (trashed ? 500 : 1));
  const size = Math.round(w * h * (format === 'png' ? 0.55 : format === 'webp' ? 0.12 : 0.22) * (0.7 + ((i * 37) % 60) / 100));
  const sha = Array.from({ length: 64 }, (_, k) => '0123456789abcdef'[(id.charCodeAt(k % 20) * (k + 7)) % 16]).join('');
  return {
    id, filename: name, thumb: `assets/fixtures/${file}`, format, ext, width: w, height: h, size, frames,
    source, folderId, sha256: sha, createdAt: NOW - minutesAgo * MIN,
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
  allowedFormats: ['jpeg', 'png', 'gif', 'webp', 'avif'],
  dedupe: true,
  trashRetentionDays: 30,
  auditRetentionDays: 180,
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

const AUDIT_PAGE = 10;
const AUDIT_KINDS = [
  { value: '', label: '全部操作' },
  { value: 'image', label: '图片' },
  { value: 'folder', label: '文件夹' },
  { value: 'token', label: '令牌' },
  { value: 'settings', label: '设置' },
  { value: 'system', label: '系统' },
  { value: 'session', label: '登录' },
];
const SX = 'ShareX 工作机（#2）';
const PG = 'PicGo 笔记本（#1）';
// [minutesAgo, actor, action, target, kind]
const AUDIT = [
  [8, SX, '上传', 'screenshot-2026-11-08-152210.png', 'image'],
  [24, PG, '上传', 'handler-retry.png', 'image'],
  [50, 'admin', '移动', '3 张图片', 'image'],
  [3 * 60, 'admin', '重命名', 'IMG_4821.jpg', 'image'],
  [5 * 60, 'admin', '创建文件夹', '旅行/2026/云南', 'folder'],
  [20 * 60, '系统', '快速检查', '1287 张图片', 'system'],
  [24 * 60, 'admin', '移到回收站', 'screenshot-old-draft.png', 'image'],
  [26 * 60, 'admin', '重命名文件夹', '博客/封面', 'folder'],
  [2 * 1440, 'admin', '修改设置', '回收站保留时间', 'settings'],
  [2 * 1440 + 90, PG, '上传', 'latency-p95.png', 'image'],
  [3 * 1440, 'admin', '移动文件夹', '品牌', 'folder'],
  [4 * 1440, SX, '上传', 'mobile-login.png', 'image'],
  [4 * 1440 + 40, 'admin', '登录', 'admin', 'session'],
  [5 * 1440, 'admin', '删除文件夹', '临时', 'folder'],
  [6 * 1440, 'admin', '恢复', 'typo-version.png', 'image'],
  [9 * 1440, 'admin', '创建快照', '20261030T030000Z', 'system'],
  [10 * 1440, 'admin', '创建令牌', '博客构建', 'token'],
  [12 * 1440, PG, '上传', 'DSC_0112.jpg', 'image'],
  [12 * 1440 + 30, PG, '上传', 'DSC_0098.jpg', 'image'],
  [14 * 1440, 'admin', '登录', 'admin', 'session'],
  [16 * 1440, '系统', '完整检查', '1287 张图片', 'system'],
  [20 * 1440, 'admin', '永久删除', 'blurry-night.jpg', 'image'],
  [30 * 1440, 'admin', '吊销令牌', '旧手机', 'token'],
  [41 * 1440, 'admin', '修改设置', '单文件上限', 'settings'],
  [62 * 1440, 'admin', '创建令牌', 'PicGo 笔记本', 'token'],
  [90 * 1440, 'admin', '创建令牌', '旧手机', 'token'],
  [120 * 1440, 'admin', '创建令牌', '备份脚本', 'token'],
].map(([m, actor, action, target, kind]) => ({ at: NOW - m * MIN, actor, action, target, kind }));
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
const EMPTY_FILTERS = { q: '', formats: [], date: '', sources: [] };

function applyFilters(list, f, sort) {
  const q = f.q.trim().toLowerCase();
  const out = list.filter((im) => {
    if (q && !(im.filename.toLowerCase().includes(q) || im.id.startsWith(q) || im.sha256.startsWith(q))) return false;
    if (f.formats.length && !f.formats.includes(im.format)) return false;
    if (f.date) {
      const r = DATE_RANGES.find((d) => d.key === f.date);
      if (r?.since && im.createdAt < r.since()) return false;
      if (r?.until && im.createdAt >= r.until()) return false;
    }
    if (f.sources.length && !f.sources.includes(im.source)) return false;
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
  return f.formats.length + (f.date ? 1 : 0) + f.sources.length;
}

// ---------------------------------------------------------------- folders
const ROOT_LABEL = '图片';
const byName = (a, b) => a.name.localeCompare(b.name, 'zh-CN');
const childrenOf = (folders, parentId) => folders.filter((f) => (f.parentId ?? null) === parentId).sort(byName);
const folderById = (folders, id) => folders.find((f) => f.id === id);
function folderChain(folders, id) {
  const out = [];
  for (let f = folderById(folders, id); f; f = folderById(folders, f.parentId)) out.unshift(f);
  return out;
}
const folderDepth = (folders, id) => (id == null ? 0 : folderChain(folders, id).length);
function folderLabel(folders, id) {
  const chain = folderChain(folders, id);
  return [ROOT_LABEL, ...chain.map((f) => f.name)].join(' / ');
}
function descendantIds(folders, id) {
  const out = new Set();
  const walk = (p) => childrenOf(folders, p).forEach((f) => (out.add(f.id), walk(f.id)));
  walk(id);
  return out;
}
function subtreeHeight(folders, id) {
  const kids = childrenOf(folders, id);
  return 1 + (kids.length ? Math.max(...kids.map((k) => subtreeHeight(folders, k.id))) : 0);
}
/** Moving `id` under `targetId` (null = root) must keep the deepest descendant at depth <= MAX_DEPTH. */
const fitsUnder = (folders, id, targetId) => folderDepth(folders, targetId) + subtreeHeight(folders, id) <= MAX_DEPTH;
/** Names -> folder id. null = root, undefined = no such folder. */
function resolvePath(folders, names) {
  let parent = null;
  for (const n of names) {
    const f = childrenOf(folders, parent).find((x) => x.name === n);
    if (!f) return undefined;
    parent = f.id;
  }
  return parent;
}
function libraryHash(folders, id) {
  const names = folderChain(folders, id).map((f) => encodeURIComponent(f.name));
  return '#/library' + names.map((n) => '/' + n).join('');
}
function folderNameError(folders, parentId, name, selfId) {
  const n = name.trim();
  if (!n) return '请输入名称';
  if (n.length > 64) return '名称最多 64 个字符';
  if (/[\\/]/.test(n)) return '名称不能包含 / 或 \\';
  if (childrenOf(folders, parentId ?? null).some((f) => f.name === n && f.id !== selfId)) return '已有同名文件夹';
  return '';
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
  { key: 'manage', label: '管理' },
];

Object.assign(window, {
  NOW, MIN, HOUR, DAY, IMG_HOST, TOKENS, SOURCES, SETTINGS, STATUS, AUDIT, FORMAT_LABEL, LINK_FORMATS, SORTS, SCOPES,
  DATE_RANGES, EMPTY_FILTERS, initialImages, makeLinks, fakeId, fmtBytes, fmtDate, fmtRel, sourceLabel, purgeIn,
  applyFilters, filterCount, FOLDERS, MAX_DEPTH, ROOT_LABEL, AUDIT_KINDS, AUDIT_PAGE, childrenOf, folderById, folderChain,
  folderDepth, folderLabel, descendantIds, subtreeHeight, fitsUnder, resolvePath, libraryHash, folderNameError,
});
