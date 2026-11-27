// Synthetic fixtures and helpers for the revision 8 candidate. No real files, hosts or tokens.
(function () {
  const NOW = new Date('2026-11-26T15:30:00+08:00').getTime();
  const MIN = 60000, HOUR = 60 * MIN, DAY = 24 * HOUR;
  const HOST = 'https://img.example.com';
  const FIX = '../assets/fixtures/';

  const PICS = {
    'dashboard-light.png': [1920, 1200, 'png', 412000],
    'dashboard-dark.png': [1920, 1200, 'png', 398000],
    'settings-panel.png': [1440, 960, 'png', 221000],
    'mobile-login.png': [1170, 2340, 'png', 305000],
    'code-snippet.png': [1600, 1000, 'png', 168000],
    'bar-chart.png': [1200, 750, 'png', 74000],
    'line-chart.png': [1600, 900, 'png', 96000],
    'architecture.png': [1680, 980, 'png', 188000],
    'mountain-dusk.jpg': [4032, 2688, 'jpeg', 3480000],
    'lake-morning.jpg': [3024, 2016, 'jpeg', 2210000],
    'desert-noon.jpg': [2268, 3402, 'jpeg', 1960000],
    'city-night.jpg': [3840, 2160, 'jpeg', 2870000],
    'receipt-scan.jpg': [1240, 1984, 'jpeg', 640000],
    'logo-mark.png': [512, 512, 'png', 18000],
    'avatar.png': [400, 400, 'png', 42000],
    'banner-wide.webp': [1800, 600, 'webp', 118000],
  };

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
  ];

  const SOURCES = { web: '网页', 'token:1': 'PicGo 笔记本', 'token:2': 'ShareX 工作机', cli: '命令行' };

  // [fixture, filename, minutesAgo, source, folderId, frames]
  const ROWS = [
    ['dashboard-light.png', 'screenshot-2026-11-26-152210.png', 8, 'token:2', 'shots', 1],
    ['code-snippet.png', 'handler-retry.png', 24, 'token:1', null, 1],
    ['architecture.png', '上传流程-v3-最终版-评审后修改.png', 70, 'web', 'architecture', 1],
    ['mountain-dusk.jpg', 'IMG_4821.jpg', 150, 'web', 'yunnan', 1],
    ['bar-chart.png', 'weekly-traffic.png', 300, 'token:1', null, 1],
    ['banner-wide.webp', 'post-cover-dark.webp', 1440 + 40, 'web', 'covers', 1],
    ['dashboard-dark.png', 'screenshot-2026-11-25-221503.png', 1440 + 300, 'token:2', 'shots', 1],
    ['receipt-scan.jpg', 'receipt-2026-11.jpg', 1440 + 600, 'web', null, 1],
    ['line-chart.png', 'latency-p95.png', 2 * 1440 + 100, 'token:1', null, 1],
    ['lake-morning.jpg', 'IMG_4790.jpg', 3 * 1440, 'web', null, 1],
    ['logo-mark.png', 'logo-mark.png', 4 * 1440, 'cli', 'brand', 1],
    ['settings-panel.png', 'settings-before.png', 5 * 1440, 'token:2', null, 1],
    ['mobile-login.png', 'mobile-login.png', 6 * 1440, 'token:2', null, 1],
    ['avatar.png', 'avatar-2026.png', 9 * 1440, 'web', 'brand', 1],
    ['city-night.jpg', 'DSC_0112.jpg', 12 * 1440, 'web', 'yunnan', 1],
    ['desert-noon.jpg', 'DSC_0098.jpg', 12 * 1440 + 20, 'web', null, 1],
    ['code-snippet.png', 'loading-spinner.gif', 22 * 1440, 'token:1', 'blog', 24],
    ['architecture.png', '数据目录结构.png', 27 * 1440, 'web', 'architecture', 1],
  ];

  const B32 = '0123456789abcdefghjkmnpqrstvwxyz';
  function fakeId(seed) {
    let s = Math.imul(seed + 97, 2654435761) >>> 0 || 1, out = '';
    for (let i = 0; i < 20; i++) { s ^= s << 13; s ^= s >>> 17; s ^= s << 5; s >>>= 0; out += B32[s % 32]; }
    return out;
  }
  function hex(seed) {
    let s = Math.imul(seed + 31, 2246822519) >>> 0 || 1, out = '';
    while (out.length < 64) { s ^= s << 13; s ^= s >>> 17; s ^= s << 5; s >>>= 0; out += (s % 16).toString(16); }
    return out;
  }
  const FORMAT = { jpeg: 'JPEG', png: 'PNG', gif: 'GIF', webp: 'WebP', avif: 'AVIF' };

  const IMAGES = ROWS.map((r, i) => {
    const [file, name, ago, source, folderId, frames] = r;
    const [w, h, fmt, size] = PICS[file];
    const id = fakeId(i + 1);
    const ext = name.split('.').pop();
    const url = `${HOST}/i/${id}.${ext}`;
    return {
      id, src: FIX + file, name, w, h, fmt: frames > 1 ? 'gif' : fmt, size: size + i * 1311, source, folderId, frames,
      createdAt: NOW - ago * MIN,
      sha: hex(i + 7),
      links: { url, markdown: `![${name}](${url})`, html: `<img src="${url}" alt="${name}">`, bbcode: `[img]${url}[/img]` },
    };
  });

  const TRASH = [
    { ...IMAGES[6], id: fakeId(201), name: 'screenshot-old-draft.png', trashedDays: 2, createdAt: NOW - 5 * DAY },
    { ...IMAGES[1], id: fakeId(202), name: 'typo-version.png', trashedDays: 26, createdAt: NOW - 31 * DAY },
    { ...IMAGES[14], id: fakeId(203), name: 'blurry-night.jpg', trashedDays: 11, createdAt: NOW - 14 * DAY },
  ];

  const TOKENS = [
    { name: 'PicGo 笔记本', prefix: 'pb_7KQ2M', scopes: ['上传'], last: NOW - 2 * HOUR, exp: null },
    { name: 'ShareX 工作机', prefix: 'pb_X4D9R', scopes: ['上传'], last: NOW - 8 * MIN, exp: NOW + 60 * DAY },
    { name: '博客构建', prefix: 'pb_M2V6C', scopes: ['上传', '管理'], last: null, exp: NOW + 20 * DAY },
  ];


  // Status, audit and an upload batch (revision 8).
  const STATUS = {
    imagesBytes: IMAGES.reduce((a, i) => a + i.size, 0), dbBytes: 1468006, diskTotal: 512 * 1024 ** 3, diskFree: 182 * 1024 ** 3,
    lastQuick: NOW - 3 * MIN, lastFull: NOW - 6 * DAY, lastSnapshot: NOW - 9 * DAY,
    version: '0.9.2', commit: 'a41c7e9', goVersion: 'go1.25.4', sqliteVersion: '3.50.4', uptime: '12 天 4 小时', dataDir: '/var/lib/picberth',
  };
  const AUDIT_KINDS = [['', '全部操作'], ['image', '图片'], ['folder', '文件夹'], ['token', '令牌'], ['settings', '设置'], ['system', '系统'], ['login', '登录']];
  const AUDIT = [
    [8, 'ShareX 工作机', '上传', 'screenshot-2026-11-26-152210.png', 'image'],
    [24, 'PicGo 笔记本', '上传', 'handler-retry.png', 'image'],
    [70, 'admin', '上传', '上传流程-v3-最终版-评审后修改.png', 'image'],
    [95, 'admin', '移动', 'IMG_4821.jpg', 'image'],
    [180, 'admin', '登录', '127.0.0.1', 'login'],
    [1440 + 40, 'admin', '新建文件夹', '博客 / 封面', 'folder'],
    [1440 + 200, 'admin', '移到回收站', 'screenshot-old-draft.png', 'image'],
    [2 * 1440, 'admin', '修改设置', '单文件上限', 'settings'],
    [3 * 1440, 'admin', '新建令牌', '博客构建', 'token'],
    [6 * 1440, 'system', '完整检查', '正常', 'system'],
    [9 * 1440, 'admin', '创建快照', '', 'system'],
  ].map(([m, actor, action, target, kind]) => ({ at: NOW - m * MIN, actor, action, target, kind }));
  // One batch in progress: two done, one duplicate, one uploading, one waiting, two failed.
  const BATCH = [
    { img: IMAGES[0], st: 'done' },
    { img: IMAGES[1], st: 'done' },
    { img: IMAGES[3], st: 'dup' },
    { img: { ...IMAGES[9], name: 'IMG_5102.jpg' }, st: 'up', p: 62 },
    { img: { ...IMAGES[14], name: 'IMG_5103.jpg' }, st: 'wait' },
    { img: { ...IMAGES[11], name: 'scan-large.png', size: 31457280 }, st: 'big' },
    { img: { ...IMAGES[7], name: 'scan-raw.tiff' }, st: 'fmt' },
  ];
  // The same batch after the session ended part-way and the administrator signed in again: the three
  // uploads the server refused can be retried, and the files that were still waiting go on by themselves.
  const BATCH_RESUMED = [
    { img: IMAGES[0], st: 'done' },
    { img: IMAGES[1], st: 'done' },
    { img: { ...IMAGES[9], name: 'IMG_5102.jpg' }, st: 'auth' },
    { img: { ...IMAGES[14], name: 'IMG_5103.jpg' }, st: 'auth' },
    { img: { ...IMAGES[11], name: 'IMG_5104.jpg' }, st: 'auth' },
    { img: { ...IMAGES[7], name: 'IMG_5105.jpg' }, st: 'up', p: 18 },
    { img: { ...IMAGES[5], name: 'IMG_5106.jpg' }, st: 'wait' },
  ];

  // ---------------------------------------------------------------- helpers
  const folderById = Object.fromEntries(FOLDERS.map((f) => [f.id, f]));
  function folderPath(id) {
    const out = [];
    for (let f = folderById[id]; f; f = folderById[f.parentId]) out.unshift(f);
    return out;
  }
  function folderLabel(id) { return id ? folderPath(id).map((f) => f.name).join(' / ') : '图片'; }
  function children(pid) { return FOLDERS.filter((f) => f.parentId === pid); }
  function directCount(id) { return IMAGES.filter((i) => i.folderId === id).length; }
  function inFolder(id) { return IMAGES.filter((i) => i.folderId === id); }
  function size(b) {
    if (b < 1024) return `${b} B`;
    if (b < 1024 * 1024) return `${Math.round(b / 1024)} KB`;
    if (b < 1024 ** 3) return `${(b / 1024 / 1024).toFixed(1)} MB`;
    return `${(b / 1024 ** 3).toFixed(1)} GB`;
  }
  function ago(t) {
    const d = NOW - t;
    if (d < MIN) return '刚刚';
    if (d < HOUR) return `${Math.round(d / MIN)} 分钟前`;
    if (d < DAY) return `${Math.round(d / HOUR)} 小时前`;
    if (d < 30 * DAY) return `${Math.round(d / DAY)} 天前`;
    const x = new Date(t);
    return `${x.getMonth() + 1} 月 ${x.getDate()} 日`;
  }
  function stamp(t) {
    const x = new Date(t), p = (n) => String(n).padStart(2, '0');
    return `${x.getFullYear()}-${p(x.getMonth() + 1)}-${p(x.getDate())} ${p(x.getHours())}:${p(x.getMinutes())}`;
  }
  function dayGroup(t) {
    const start = new Date(NOW); start.setHours(0, 0, 0, 0);
    const s = start.getTime();
    if (t >= s) return '今天';
    if (t >= s - DAY) return '昨天';
    const x = new Date(t);
    return `${x.getMonth() + 1} 月 ${x.getDate()} 日`;
  }
  // Middle truncation by approximate width (CJK ≈ 13 px, Latin ≈ 7 px) keeps the start and the extension.
  const cw = (c) => (/[⺀-￿]/.test(c) ? 13 : 7);
  const wOf = (str) => [...str].reduce((a, c) => a + cw(c), 0);
  function midName(name, px) {
    if (wOf(name) <= px) return name;
    const dot = name.lastIndexOf('.');
    const ext = dot > 0 ? name.slice(dot) : '';
    const chars = [...name.slice(0, dot > 0 ? dot : name.length)];
    let budget = px - wOf(ext) - 13, head = '', tail = '';
    let i = 0, j = chars.length - 1;
    while (i <= j && budget > 0) {
      if (wOf(head) <= wOf(tail) * 1.6) { budget -= cw(chars[i]); if (budget >= 0) head += chars[i++]; }
      else { budget -= cw(chars[j]); if (budget >= 0) tail = chars[j--] + tail; }
    }
    return head + '…' + tail + ext;
  }
  const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

  // ---------------------------------------------------------------- icons (16 px, 1.5 stroke)
  const P = {
    image: '<rect x="2.5" y="2.5" width="11" height="11" rx="2"/><circle cx="6" cy="6" r="1.2"/><path d="m13.5 10-3-3-6.5 6.5"/>',
    folder: '<path d="M2 4.5c0-.8.7-1.5 1.5-1.5h2.6l1.5 1.7h4.9c.8 0 1.5.7 1.5 1.5v5.3c0 .8-.7 1.5-1.5 1.5h-9c-.8 0-1.5-.7-1.5-1.5Z"/>',
    trash: '<path d="M2.5 4h11M6 4V2.8c0-.4.3-.8.8-.8h2.4c.5 0 .8.4.8.8V4M4 4l.6 8.6c0 .8.7 1.4 1.5 1.4h3.8c.8 0 1.5-.6 1.5-1.4L12 4"/>',
    key: '<circle cx="5.5" cy="10.5" r="3"/><path d="m7.6 8.4 5.9-5.9M11 5l1.5 1.5"/>',
    sliders: '<path d="M2.5 5h6M11.5 5h2M2.5 11h2M7.5 11h6"/><circle cx="10" cy="5" r="1.5"/><circle cx="6" cy="11" r="1.5"/>',
    pulse: '<path d="M1.5 8h3l2-4.5 3 9 2-4.5h3"/>',
    gear: '<circle cx="8" cy="8" r="2"/><path d="M8 1.8v1.6M8 12.6v1.6M3.6 3.6l1.1 1.1M11.3 11.3l1.1 1.1M1.8 8h1.6M12.6 8h1.6M3.6 12.4l1.1-1.1M11.3 4.7l1.1-1.1"/>',
    upload: '<path d="M8 10.5V2.5M4.8 5.5 8 2.3l3.2 3.2M2.5 10.5v1.5c0 .8.7 1.5 1.5 1.5h8c.8 0 1.5-.7 1.5-1.5v-1.5"/>',
    search: '<circle cx="7" cy="7" r="4.5"/><path d="m10.5 10.5 3 3"/>',
    filter: '<path d="M2.5 4h11M4.5 8h7M6.5 12h3"/>',
    sort: '<path d="M5 2.5v11M2.5 11 5 13.5 7.5 11M11 13.5v-11M8.5 5 11 2.5 13.5 5"/>',
    grid: '<rect x="2.5" y="2.5" width="4.5" height="4.5" rx="1"/><rect x="9" y="2.5" width="4.5" height="4.5" rx="1"/><rect x="2.5" y="9" width="4.5" height="4.5" rx="1"/><rect x="9" y="9" width="4.5" height="4.5" rx="1"/>',
    list: '<path d="M5.5 4h8M5.5 8h8M5.5 12h8M2.5 4h.01M2.5 8h.01M2.5 12h.01"/>',
    copy: '<rect x="5.5" y="5.5" width="8" height="8" rx="1.5"/><path d="M10.5 5.5V4c0-.8-.7-1.5-1.5-1.5H4c-.8 0-1.5.7-1.5 1.5v5c0 .8.7 1.5 1.5 1.5h1.5"/>',
    check: '<path d="m3 8.5 3 3 7-7"/>',
    x: '<path d="m4 4 8 8M12 4l-8 8"/>',
    right: '<path d="m6 3.5 4.5 4.5L6 12.5"/>',
    left: '<path d="M10 3.5 5.5 8l4.5 4.5"/>',
    down: '<path d="M3.5 6 8 10.5 12.5 6"/>',
    more: '<path d="M3.5 8h.01M8 8h.01M12.5 8h.01" stroke-width="2.2"/>',
    plus: '<path d="M8 3v10M3 8h10"/>',
    download: '<path d="M8 2.5v8M4.8 7.5 8 10.7l3.2-3.2M2.5 13.5h11"/>',
    move: '<path d="M2 4.5c0-.8.7-1.5 1.5-1.5h2.6l1.5 1.7h4.9c.8 0 1.5.7 1.5 1.5v5.3c0 .8-.7 1.5-1.5 1.5h-9c-.8 0-1.5-.7-1.5-1.5Z"/><path d="M6 9h4M8.5 7.5 10 9l-1.5 1.5"/>',
    clock: '<circle cx="8" cy="8" r="5.5"/><path d="M8 5v3l2 1.5"/>',
    expand: '<path d="M9.5 2.5h4v4M6.5 13.5h-4v-4M13.5 2.5 9 7M2.5 13.5 7 9"/>',
    restore: '<path d="M3 8a5 5 0 1 0 1.5-3.5M3 2.5v2.5h2.5"/>',
    menu: '<path d="M2.5 4.5h11M2.5 8h11M2.5 11.5h11"/>',
    home: '<path d="M2.5 7 8 2.5 13.5 7v5.5c0 .6-.4 1-1 1h-9c-.6 0-1-.4-1-1Z"/><path d="M6.5 13.5V9.5h3v4"/>',
    inbox: '<path d="M2.5 9.5 4.3 3.6c.2-.6.7-1.1 1.4-1.1h4.6c.7 0 1.2.5 1.4 1.1l1.8 5.9V12c0 .8-.7 1.5-1.5 1.5H4c-.8 0-1.5-.7-1.5-1.5Z"/><path d="M2.5 9.5h3l1 1.5h3l1-1.5h3"/>',
    disk: '<rect x="2" y="3.5" width="12" height="9" rx="2"/><path d="M5 9.5h.01M8 9.5h3"/>',
    refresh: '<path d="M13 8a5 5 0 1 1-1.5-3.5M13 2.5V5h-2.5"/>',
    alert: '<path d="M8 2.5 14 13H2Z"/><path d="M8 6.5v3M8 11.2h.01"/>',
    external: '<path d="M9.5 2.5h4v4M13.5 2.5 8 8M11.5 9.5v3c0 .6-.4 1-1 1h-7c-.6 0-1-.4-1-1v-7c0-.6.4-1 1-1h3"/>',
    link: '<path d="M7 9a2.5 2.5 0 0 0 3.5 0l2-2A2.5 2.5 0 0 0 9 3.5l-.7.7M9 7a2.5 2.5 0 0 0-3.5 0l-2 2A2.5 2.5 0 0 0 7 12.5l.7-.7"/>',
  };
  function icon(name, cls = '') { return `<svg class="icon ${cls}" viewBox="0 0 16 16" aria-hidden="true">${P[name] || ''}</svg>`; }

  // ---------------------------------------------------------------- page params and shared behaviour
  const params = new URLSearchParams(location.search);
  const theme = params.get('theme') || (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
  document.documentElement.dataset.theme = theme;

  function toast(text) {
    let el = document.querySelector('.x-toast');
    if (!el) { el = document.createElement('div'); el.className = 'x-toast'; el.setAttribute('role', 'status'); document.body.appendChild(el); }
    el.innerHTML = `${icon('check')}<span>${esc(text)}</span>`;
    el.dataset.on = 'true';
    clearTimeout(el._t);
    el._t = setTimeout(() => (el.dataset.on = 'false'), 1800);
  }

  // Justified rows: every image keeps its proportions, rows fill the width.
  function justify(items, width, target, gap) {
    const rows = []; let row = [], sum = 0;
    for (const it of items) {
      row.push(it); sum += it.w / it.h;
      const h = (width - gap * (row.length - 1)) / sum;
      if (h < target) { rows.push({ items: row, h }); row = []; sum = 0; }
    }
    if (row.length) rows.push({ items: row, h: target, last: true });
    return rows;
  }

  window.PB = { STATUS, AUDIT, AUDIT_KINDS, BATCH, BATCH_RESUMED, NOW, MIN, HOUR, DAY, FOLDERS, IMAGES, TRASH, TOKENS, SOURCES, FORMAT, folderPath, folderLabel, children, directCount, inFolder, size, ago, stamp, dayGroup, midName, esc, icon, params, theme, toast, justify };
})();
