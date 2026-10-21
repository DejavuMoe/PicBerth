// App state, the three layout directions and the prototype options panel.

const PARAMS = new URLSearchParams(location.search);
const EMBED = PARAMS.has('embed');

const NAV = [
  { key: 'images', label: '全部图片', short: '图片', icon: 'images', group: '图片' },
  { key: 'trash', label: '回收站', short: '回收站', icon: 'trash', group: '图片' },
  { key: 'tokens', label: '令牌', short: '令牌', icon: 'key', group: '接入' },
  { key: 'settings', label: '设置', short: '设置', icon: 'gear', group: '系统' },
  { key: 'status', label: '状态', short: '状态', icon: 'activity', group: '系统' },
];
const DEFAULT_VIEW = { a: 'table', b: 'grid', c: 'grid' };

const SAMPLE_UPLOADS = [
  { file: 'dashboard-light.png', name: 'screenshot-2026-10-21-153012.png', size: 418000, format: 'png', width: 1920, height: 1200 },
  { file: 'line-chart.png', name: 'latency-p99.png', size: 236000, format: 'png', width: 1600, height: 900 },
  { file: 'city-night.jpg', name: 'DSC_0140.jpg', size: 31 * 1048576, format: 'jpeg', width: 6000, height: 4000 },
  { file: 'code-snippet.png', name: 'handler-retry.png', size: 188000, format: 'png', width: 1600, height: 1000, dup: true },
  { file: 'bar-chart.png', name: 'weekly-traffic-v2.png', size: 274000, format: 'png', width: 1200, height: 750, failAt: 55 },
];

let SEQ = 700;
function buildImage(u) {
  const id = fakeId(SEQ++);
  const ext = u.format === 'jpeg' ? 'jpg' : u.format;
  const url = `https://img.example.com/i/${id}.${ext}`;
  return {
    id,
    filename: u.name,
    thumb: u.thumb,
    format: u.format,
    mime: MIME[u.format],
    ext,
    width: u.width,
    height: u.height,
    size: u.size,
    frames: 1,
    visibility: u.vis,
    hasGps: false,
    source: 'web',
    tags: [],
    sha256: (id + id + id + id).slice(0, 64).replace(/[^0-9a-f]/g, (c) => '0123456789abcdef'[c.charCodeAt(0) % 16]),
    createdAt: NOW,
    trashedAt: null,
    state: 'active',
    url,
    links: { url, markdown: `![${u.name}](${url})`, html: `<img src="${url}" alt="${u.name}">`, bbcode: `[img]${url}[/img]` },
  };
}

const MIME_FORMAT = { 'image/jpeg': 'jpeg', 'image/png': 'png', 'image/gif': 'gif', 'image/webp': 'webp', 'image/avif': 'avif' };
function readFiles(files) {
  return Promise.all(
    [...files].map(
      (f) =>
        new Promise((res) => {
          const format = MIME_FORMAT[f.type];
          const thumb = format ? URL.createObjectURL(f) : '';
          const meta = { name: f.name || 'image.png', size: f.size, format, thumb, width: 0, height: 0 };
          if (!format) return res(meta);
          const im = new Image();
          im.onload = () => res({ ...meta, width: im.naturalWidth, height: im.naturalHeight });
          im.onerror = () => res(meta);
          im.src = thumb;
        }),
    ),
  );
}

const isTyping = (e) => !!e.target.closest?.('input, textarea, select, [contenteditable="true"]');
const isActiveUpload = (u) => u.status === 'queued' || u.status === 'uploading' || u.status === 'processing';

function App() {
  const [dir, setDirRaw] = React.useState(PARAMS.get('d') || 'a');
  const [theme, setTheme] = React.useState(PARAMS.get('theme') || 'light');
  const [ext, setExt] = React.useState(PARAMS.get('ext') !== '0');
  const [dataState, setDataStateRaw] = React.useState(PARAMS.get('state') || 'ok');
  const [route, setRoute] = React.useState(PARAMS.get('r') || 'images');
  const [images, setImages] = React.useState(() => (PARAMS.get('state') === 'empty' ? [] : initialImages()));
  const [tokens, setTokens] = React.useState(TOKENS);
  const [settings, setSettings] = React.useState(SETTINGS);
  const [filters, setFilters] = React.useState(EMPTY_FILTERS);
  const [sort, setSort] = React.useState('new');
  const [view, setView] = React.useState(PARAMS.get('view') || DEFAULT_VIEW[PARAMS.get('d') || 'a']);
  const [sel, setSel] = React.useState(() => new Set());
  const [selectMode, setSelectMode] = React.useState(false);
  const [activeId, setActiveId] = React.useState(null);
  const [linkFormat, setLinkFormat] = React.useState('url');
  const [uploads, setUploads] = React.useState([]);
  const [uploadVis, setUploadVis] = React.useState(SETTINGS.defaultVisibility);
  const [uploadOpen, setUploadOpen] = React.useState(false);
  const [toasts, setToasts] = React.useState([]);
  const [purgeIds, setPurgeIds] = React.useState(null);
  const [dragging, setDragging] = React.useState(false);
  const [navOpen, setNavOpen] = React.useState(false);
  const [filterSheet, setFilterSheet] = React.useState(false);

  const isMobile = useMedia('(max-width: 720px)');
  const wideB = useMedia('(min-width: 1181px)');
  const anchor = React.useRef(null);
  const suppress = React.useRef(false);
  const fileInput = React.useRef(null);
  const searchRef = React.useRef(null);
  const uploadsRef = React.useRef(uploads);
  uploadsRef.current = uploads;

  React.useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  // ---------------------------------------------------------------- derived lists
  const byId = React.useMemo(() => Object.fromEntries(images.map((i) => [i.id, i])), [images]);
  const inLibrary = route === 'images' || route === 'trash';
  const base = React.useMemo(() => images.filter((i) => i.state === (route === 'trash' ? 'trashed' : 'active')), [images, route]);
  const items = React.useMemo(() => applyFilters(base, filters, sort, ext), [base, filters, sort, ext]);
  const knownTags = React.useMemo(() => allTags(images).map(([t]) => t), [images]);
  const active = activeId ? byId[activeId] : null;

  // ---------------------------------------------------------------- toasts
  const toast = React.useCallback((text, action) => {
    const id = Math.random().toString(36).slice(2);
    setToasts((t) => [...t.slice(-2), { id, text, action }]);
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), action ? 6000 : 3200);
  }, []);
  const dismiss = (id) => setToasts((t) => t.filter((x) => x.id !== id));

  // ---------------------------------------------------------------- selection
  const clearSel = () => {
    setSel(new Set());
    setSelectMode(false);
    anchor.current = null;
  };
  const toggle = (id, shift) => {
    setSel((prev) => {
      const next = new Set(prev);
      if (shift && anchor.current) {
        const ids = items.map((i) => i.id);
        const a = ids.indexOf(anchor.current);
        const b = ids.indexOf(id);
        if (a >= 0 && b >= 0) {
          for (let i = Math.min(a, b); i <= Math.max(a, b); i++) next.add(ids[i]);
          return next;
        }
      }
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
    anchor.current = id;
  };
  const open = (id) => {
    setActiveId(id);
    if (dir === 'b') setSel(new Set());
  };
  const clickItem = (id, e) => {
    if (suppress.current) {
      suppress.current = false;
      return;
    }
    if (e.metaKey || e.ctrlKey) return toggle(id);
    if (e.shiftKey) return toggle(id, true);
    if (sel.size || selectMode) return toggle(id);
    open(id);
  };
  const neighbors = (id) => {
    const i = items.findIndex((x) => x.id === id);
    return [items[i - 1], items[i + 1]];
  };

  // ---------------------------------------------------------------- actions
  const patch = (ids, fn) => setImages((list) => list.map((im) => (ids.includes(im.id) ? fn(im) : im)));
  const countText = (ids, one) => (ids.length === 1 ? one : `${ids.length} 张图片`);
  const act = {
    trash(ids) {
      patch(ids, (im) => ({ ...im, state: 'trashed', trashedAt: NOW }));
      clearSel();
      if (ids.includes(activeId)) setActiveId(null);
      toast(`已将${countText(ids, '图片')}移到回收站`, { label: '撤销', run: () => patch(ids, (im) => ({ ...im, state: 'active', trashedAt: null })) });
    },
    restore(ids) {
      const prev = Object.fromEntries(ids.map((id) => [id, byId[id]?.trashedAt]));
      patch(ids, (im) => ({ ...im, state: 'active', trashedAt: null }));
      clearSel();
      if (ids.includes(activeId)) setActiveId(null);
      toast(`已恢复${countText(ids, '图片')}`, { label: '撤销', run: () => patch(ids, (im) => ({ ...im, state: 'trashed', trashedAt: prev[im.id] })) });
    },
    purge(ids) {
      setPurgeIds(ids);
    },
    setVis(ids, v) {
      const prev = Object.fromEntries(ids.map((id) => [id, byId[id]?.visibility]));
      patch(ids, (im) => ({ ...im, visibility: v }));
      toast(`已将${countText(ids, '图片')}设为${v === 'public' ? '公开' : '私有'}`, { label: '撤销', run: () => patch(ids, (im) => ({ ...im, visibility: prev[im.id] })) });
    },
    rename(id, name) {
      patch([id], (im) => ({ ...im, filename: name }));
      toast('已重命名');
    },
    setTags(id, tags) {
      patch([id], (im) => ({ ...im, tags }));
    },
    download(im) {
      toast(`已开始下载 ${im.filename}`);
    },
  };
  const confirmPurge = () => {
    const ids = purgeIds;
    setImages((list) => list.filter((im) => !ids.includes(im.id)));
    setPurgeIds(null);
    clearSel();
    if (ids.includes(activeId)) setActiveId(null);
    toast(`已永久删除${countText(ids, '图片')}`);
  };

  // ---------------------------------------------------------------- uploads (simulated transfer)
  const addUploads = (metas) => {
    if (!metas.length) return;
    const list = metas.map((m) => {
      let error = null;
      if (!m.format || !settings.allowedFormats.includes(m.format)) error = '不支持的图片格式';
      else if (m.size > settings.maxUploadMB * 1048576) error = `文件超过上限（${fmtBytes(m.size)}，上限 ${settings.maxUploadMB} MB）`;
      const dupOf = m.dup ? images.find((i) => i.filename === m.name && i.state === 'active')?.id : null;
      return { ...m, id: Math.random().toString(36).slice(2), format: m.format || 'png', vis: uploadVis, status: error ? 'error' : 'queued', error, retryable: false, progress: 0, dupOf };
    });
    setUploads((u) => [...list, ...u]);
    if (dir === 'a') setUploadOpen(true);
    if (dir === 'c' && route !== 'images') setRoute('images');
  };
  const pickFiles = () => fileInput.current?.click();
  const simulateUploads = () => addUploads(SAMPLE_UPLOADS.map((s) => ({ ...s, thumb: `assets/fixtures/${s.file}` })));
  const hasActive = uploads.some(isActiveUpload);
  const wasActive = React.useRef(false);

  React.useEffect(() => {
    if (!hasActive) return;
    const t = setInterval(() => {
      let running = uploadsRef.current.filter((u) => u.status === 'uploading' || u.status === 'processing').length;
      const created = [];
      const next = [...uploadsRef.current]
        .reverse()
        .map((u) => {
          if (u.status === 'queued' && running < 3) {
            running++;
            return { ...u, status: 'uploading', progress: 0 };
          }
          if (u.status === 'uploading') {
            const step = (6 + Math.random() * 16) * (u.size > 2 * 1048576 ? 0.5 : 1);
            const p = Math.min(100, Math.round(u.progress + step));
            if (u.failAt && !u.failed && p >= u.failAt) return { ...u, status: 'error', error: '无法连接到服务器', retryable: true, failed: true, progress: p };
            if (p >= 100) return { ...u, status: 'processing', progress: 100, wait: 2 };
            return { ...u, progress: p };
          }
          if (u.status === 'processing') {
            if (u.wait > 0) return { ...u, wait: u.wait - 1 };
            if (u.dupOf) return { ...u, status: 'duplicate', imageId: u.dupOf };
            const img = buildImage(u);
            created.push(img);
            return { ...u, status: 'done', imageId: img.id };
          }
          return u;
        })
        .reverse();
      uploadsRef.current = next;
      setUploads(next);
      if (created.length) setImages((list) => [...created.reverse(), ...list]);
    }, 220);
    return () => clearInterval(t);
  }, [hasActive]);

  React.useEffect(() => {
    if (wasActive.current && !hasActive) {
      const done = uploads.filter((u) => u.status === 'done' || u.status === 'duplicate').length;
      const failed = uploads.filter((u) => u.status === 'error').length;
      if (!(dir === 'a' && uploadOpen)) toast(failed ? `上传完成 ${done} 个，失败 ${failed} 个` : `已上传 ${done} 张图片`);
    }
    wasActive.current = hasActive;
  }, [hasActive]);

  const cancelUpload = (id) => {
    setUploads((u) => u.filter((x) => x.id !== id));
    toast('已取消上传');
  };
  const retryUpload = (id) => setUploads((u) => u.map((x) => (x.id === id ? { ...x, status: 'queued', progress: 0, error: null } : x)));
  const removeUpload = (id) => setUploads((u) => u.filter((x) => x.id !== id));
  const clearFinished = () => setUploads((u) => u.filter((x) => isActiveUpload(x) || x.status === 'error'));

  // ---------------------------------------------------------------- window-wide drop, paste and shortcuts
  React.useEffect(() => {
    let depth = 0;
    const hasFiles = (e) => [...(e.dataTransfer?.types ?? [])].includes('Files');
    const enter = (e) => {
      if (!hasFiles(e)) return;
      e.preventDefault();
      depth++;
      setDragging(true);
    };
    const over = (e) => hasFiles(e) && e.preventDefault();
    const leave = (e) => {
      if (!hasFiles(e)) return;
      depth = Math.max(0, depth - 1);
      if (!depth) setDragging(false);
    };
    const drop = (e) => {
      if (!hasFiles(e)) return;
      e.preventDefault();
      depth = 0;
      setDragging(false);
      readFiles(e.dataTransfer.files).then(addUploadsRef.current);
    };
    const paste = (e) => {
      if (isTyping(e)) return;
      const files = e.clipboardData?.files;
      if (files?.length) readFiles(files).then(addUploadsRef.current);
    };
    window.addEventListener('dragenter', enter);
    window.addEventListener('dragover', over);
    window.addEventListener('dragleave', leave);
    window.addEventListener('drop', drop);
    window.addEventListener('paste', paste);
    return () => {
      window.removeEventListener('dragenter', enter);
      window.removeEventListener('dragover', over);
      window.removeEventListener('dragleave', leave);
      window.removeEventListener('drop', drop);
      window.removeEventListener('paste', paste);
    };
  }, []);
  const addUploadsRef = React.useRef(addUploads);
  addUploadsRef.current = addUploads;

  const keyRef = React.useRef(null);
  keyRef.current = (e) => {
    if (isTyping(e) || e.metaKey || e.ctrlKey || e.altKey) return;
    if (document.querySelector('.modal, .drawer, .sheet') && e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return;
    if (e.key === '/' && inLibrary) {
      e.preventDefault();
      searchRef.current?.focus();
    } else if (e.key === 'u' || e.key === 'U') {
      pickFiles();
    } else if (e.key === 'Escape') {
      if (sel.size || selectMode) clearSel();
      else if (activeId) setActiveId(null);
    } else if ((e.key === 'ArrowLeft' || e.key === 'ArrowRight') && active) {
      const [p, n] = neighbors(active.id);
      const t = e.key === 'ArrowLeft' ? p : n;
      if (t) setActiveId(t.id);
    }
  };
  React.useEffect(() => {
    const k = (e) => keyRef.current(e);
    document.addEventListener('keydown', k);
    return () => document.removeEventListener('keydown', k);
  }, []);

  // ---------------------------------------------------------------- navigation and prototype options
  const go = (r) => {
    setRoute(r);
    clearSel();
    setActiveId(null);
    setNavOpen(false);
    window.scrollTo(0, 0);
  };
  const setDir = (d) => {
    setDirRaw(d);
    setView(DEFAULT_VIEW[d]);
    setActiveId(null);
    setUploadOpen(false);
    clearSel();
  };
  const setDataState = (s) => {
    setDataStateRaw(s);
    if (s === 'empty') setImages([]);
    else if (dataState === 'empty') setImages(initialImages());
  };

  // Deep links used by the review screenshots.
  React.useEffect(() => {
    const o = PARAMS.get('open');
    if (o !== null) setActiveId(base[Number(o) || 0]?.id ?? null);
    const s = Number(PARAMS.get('sel') || 0);
    if (s) setSel(new Set(base.slice(0, s).map((i) => i.id)));
    if (PARAMS.get('up')) simulateUploads();
    if (PARAMS.get('upload')) setUploadOpen(true);
    if (PARAMS.get('filters')) setFilterSheet(true);
    if (PARAMS.get('purge')) setPurgeIds(base.slice(0, 1).map((i) => i.id));
    if (PARAMS.get('nav')) setNavOpen(true);
    if (PARAMS.get('f')) setFilters({ ...EMPTY_FILTERS, formats: ['png'], vis: 'public', tags: ext ? ['博客'] : [] });
  }, []);

  const lib = {
    images, base, items, byId, route, filters, setFilters, sort, setSort, view, setView, sel, selectMode, setSelectMode,
    activeId, open, clickItem, toggle, neighbors, selectAll: () => setSel(new Set(items.map((i) => i.id))), clearSel,
    suppressClick: () => (suppress.current = true), ext, linkFormat, setLinkFormat, act, dataState,
    retry: () => setDataState('ok'), settings, isMobile, toast, pick: pickFiles, go, knownTags,
    uploads, uploadVis, setUploadVis, cancelUpload, retryUpload, removeUpload, clearFinished,
  };
  const shared = { lib, route, go, theme, setTheme, toast, status: STATUS, images, isMobile };

  const page =
    route === 'tokens' ? (
      <TokensPage tokens={tokens} setTokens={setTokens} toast={toast} />
    ) : route === 'settings' ? (
      <SettingsPage settings={settings} setSettings={setSettings} toast={toast} />
    ) : route === 'status' ? (
      <StatusPage status={STATUS} images={images} toast={toast} />
    ) : null;

  const Shell = dir === 'b' ? ShellB : dir === 'c' ? ShellC : ShellA;
  return (
    <>
      <Shell {...shared} page={page} searchRef={searchRef} navOpen={navOpen} setNavOpen={setNavOpen} openFilters={() => setFilterSheet(true)} uploadOpen={uploadOpen} setUploadOpen={setUploadOpen} active={active} closeDetail={() => setActiveId(null)} wideB={wideB} />
      {filterSheet && <FilterSheet lib={lib} onClose={() => setFilterSheet(false)} />}
      {purgeIds && <PurgeConfirm count={purgeIds.length} onCancel={() => setPurgeIds(null)} onConfirm={confirmPurge} />}
      {dragging && <DropOverlay />}
      <input
        ref={fileInput}
        type="file"
        accept="image/jpeg,image/png,image/gif,image/webp,image/avif"
        multiple
        hidden
        onChange={(e) => {
          readFiles(e.target.files).then(addUploads);
          e.target.value = '';
        }}
      />
      <Toasts toasts={toasts} dismiss={dismiss} />
      {!EMBED && (
        <ProtoPanel dir={dir} setDir={setDir} theme={theme} setTheme={setTheme} ext={ext} setExt={setExt} dataState={dataState} setDataState={setDataState} simulate={simulateUploads} />
      )}
    </>
  );
}

// ---------------------------------------------------------------- shared chrome

function ThemeButton({ theme, setTheme }) {
  const dark = theme === 'dark';
  return (
    <button type="button" className="btn btn-ghost btn-icon" aria-label={dark ? '切换到浅色' : '切换到深色'} title={dark ? '切换到浅色' : '切换到深色'} onClick={() => setTheme(dark ? 'light' : 'dark')}>
      <Icon name={dark ? 'sun' : 'moon'} />
    </button>
  );
}

function AccountMenu({ toast }) {
  return (
    <Menu
      align="right"
      width={180}
      trigger={(open, toggle) => (
        <button type="button" className="btn btn-ghost btn-icon" aria-label="账户" aria-expanded={open} onClick={toggle}>
          <span className="avatar">A</span>
        </button>
      )}
    >
      {(close) => (
        <>
          <div className="menu-label">admin</div>
          <div className="menu-sep" />
          <MenuItem icon="logout" onClick={() => (close(), toast('已退出登录'))}>
            退出登录
          </MenuItem>
        </>
      )}
    </Menu>
  );
}

function navCount(images, key) {
  if (key === 'images') return images.filter((i) => i.state === 'active').length;
  if (key === 'trash') return images.filter((i) => i.state === 'trashed').length;
  return undefined;
}

function NavList({ route, go, images, status }) {
  let group = null;
  return NAV.map((n) => {
    const head = n.group !== group ? <div className="nav-group">{(group = n.group)}</div> : null;
    const count = navCount(images, n.key);
    return (
      <React.Fragment key={n.key}>
        {head}
        <button type="button" className="nav-item" aria-current={route === n.key ? 'page' : undefined} onClick={() => go(n.key)}>
          <Icon name={n.icon} />
          {n.label}
          {count !== undefined && <span className="count">{count}</span>}
          {n.key === 'status' && status.warnings.length > 0 && (
            <span className="count" aria-label={`${status.warnings.length} 条警告`}>
              <span className="dot" style={{ color: 'var(--warning)' }} />
            </span>
          )}
        </button>
      </React.Fragment>
    );
  });
}

function UsageMeter({ images, status }) {
  const bytes = images.reduce((s, i) => s + i.size, 0);
  const used = 1 - status.disk.freeBytes / status.disk.totalBytes;
  return (
    <div className="usage">
      <div className="usage-row">
        <span>图片占用</span>
        <span className="num">{fmtBytes(bytes)}</span>
      </div>
      <div className="progress" aria-label="磁盘已用">
        <span style={{ width: `${Math.round(used * 100)}%` }} />
      </div>
      <div className="usage-row">
        <span>磁盘剩余</span>
        <span className="num">{fmtBytes(status.disk.freeBytes)}</span>
      </div>
    </div>
  );
}

function LibraryHead({ lib, route, onUpload, compact }) {
  const total = lib.base.reduce((s, i) => s + i.size, 0);
  if (route === 'trash')
    return (
      <div className="page-head">
        <div>
          <h1>回收站</h1>
          <p>
            {lib.base.length} 张 · {fmtBytes(total)} · 保留 {lib.settings.trashRetentionDays} 天后自动永久删除
          </p>
        </div>
        {lib.base.length > 0 && (
          <div className="page-actions">
            <button type="button" className="btn btn-danger" onClick={() => lib.act.purge(lib.base.map((i) => i.id))}>
              清空回收站
            </button>
          </div>
        )}
      </div>
    );
  return (
    <div className="page-head">
      <div>
        <h1>{compact ? '图片' : '全部图片'}</h1>
        <p>
          {lib.base.length} 张 · {fmtBytes(total)}
        </p>
      </div>
      {onUpload && (
        <div className="page-actions">
          <button type="button" className="btn btn-primary" onClick={onUpload}>
            <Icon name="upload" />
            上传
          </button>
        </div>
      )}
    </div>
  );
}

// ================================================================ A — sidebar console

function ShellA({ lib, route, go, theme, setTheme, toast, status, images, page, searchRef, navOpen, setNavOpen, openFilters, uploadOpen, setUploadOpen, active, closeDetail, isMobile }) {
  const cur = NAV.find((n) => n.key === route);
  return (
    <div className="shell-a" data-screen-label="direction-a">
      <aside className="sidenav" aria-label="主导航">
        <Brand />
        <nav>
          <NavList route={route} go={go} images={images} status={status} />
        </nav>
        <UsageMeter images={images} status={status} />
      </aside>
      <div className="main-a">
        <header className="topbar-a">
          <button type="button" className="btn btn-ghost btn-icon only-mobile" aria-label="打开导航" onClick={() => setNavOpen(true)}>
            <Icon name="menu" />
          </button>
          <nav className="crumbs" aria-label="位置">
            <span className="hide-sm">PicBerth</span>
            <span className="hide-sm">/</span>
            {cur.group !== '图片' && (
              <>
                <span className="hide-sm">{cur.group}</span>
                <span className="hide-sm">/</span>
              </>
            )}
            <strong>{cur.label}</strong>
          </nav>
          <span className="grow" />
          <ThemeButton theme={theme} setTheme={setTheme} />
          <AccountMenu toast={toast} />
        </header>
        <main>
          {page ?? (
            <div className="page">
              <LibraryHead lib={lib} route={route} onUpload={() => setUploadOpen(true)} />
              <LibraryToolbar lib={lib} searchRef={searchRef} onOpenFilters={openFilters} />
              <ImageCollection lib={lib} />
            </div>
          )}
        </main>
      </div>
      {navOpen && (
        <Drawer side="left" width="272px" onClose={() => setNavOpen(false)} label="导航">
          <div className="sidenav in-drawer">
            <Brand />
            <nav>
              <NavList route={route} go={go} images={images} status={status} />
            </nav>
            <UsageMeter images={images} status={status} />
          </div>
        </Drawer>
      )}
      {active && (
        <Drawer onClose={closeDetail} width="440px" label="图片详情">
          <DetailPanel lib={lib} image={active} onClose={closeDetail} />
        </Drawer>
      )}
      {uploadOpen && <UploadDialog lib={lib} onClose={() => setUploadOpen(false)} />}
    </div>
  );
}

// ================================================================ B — facets, list, inspector

function ShellB({ lib, route, go, theme, setTheme, toast, status, images, page, searchRef, navOpen, setNavOpen, openFilters, active, closeDetail, wideB, isMobile }) {
  const [trayHidden, setTrayHidden] = React.useState(false);
  React.useEffect(() => {
    if (lib.uploads.some((u) => u.status === 'queued')) setTrayHidden(false);
  }, [lib.uploads]);
  const showInspector = !page;
  const n = activeFilterCount({ ...lib.filters, q: '' }, lib.ext);
  const narrow = useMedia('(max-width: 860px)');
  const multi = lib.sel.size > 0;
  const navTop = (
    <nav className="b-nav" aria-label="主导航">
      {NAV.slice(0, 2).map((x) => (
        <button key={x.key} type="button" className="nav-item" aria-current={route === x.key ? 'page' : undefined} onClick={() => go(x.key)}>
          <Icon name={x.icon} />
          {x.label}
          <span className="count">{navCount(images, x.key)}</span>
        </button>
      ))}
    </nav>
  );
  const navFoot = (
    <nav className="nav-foot" aria-label="管理">
      {NAV.slice(2).map((x) => (
        <button key={x.key} type="button" className="nav-item" aria-current={route === x.key ? 'page' : undefined} onClick={() => go(x.key)}>
          <Icon name={x.icon} />
          {x.label}
          {x.key === 'status' && status.warnings.length > 0 && (
            <span className="count">
              <span className="dot" style={{ color: 'var(--warning)' }} />
            </span>
          )}
        </button>
      ))}
    </nav>
  );
  return (
    <div className="shell-b" data-screen-label="direction-b">
      <header className="header-b">
        {narrow && (
          <button type="button" className="btn btn-ghost btn-icon" aria-label="打开导航" onClick={() => setNavOpen(true)}>
            <Icon name="menu" />
          </button>
        )}
        <Brand />
        {page ? <span className="grow" /> : <SearchBox lib={lib} inputRef={searchRef} placeholder={isMobile ? '搜索图片' : '搜索文件名、ID 或 SHA-256'} />}
        <button type="button" className={`btn btn-primary ${isMobile ? 'btn-icon' : ''}`} aria-label={isMobile ? '上传' : undefined} onClick={lib.pick}>
          <Icon name="upload" />
          {!isMobile && '上传'}
        </button>
        {!isMobile && <ThemeButton theme={theme} setTheme={setTheme} />}
        <AccountMenu toast={toast} />
      </header>
      <div className={`body-b ${showInspector ? '' : 'no-inspector'}`}>
        <aside className="facets">
          {navTop}
          {!page && <FacetPanel lib={lib} />}
          {navFoot}
        </aside>
        <main className="center-b">
          {page ?? (
            <>
              <div className="listhead">
                {lib.dataState === 'ok' && lib.items.length > 0 && !isMobile && (
                  <input
                    type="checkbox"
                    className="check"
                    aria-label="全选"
                    checked={multi && lib.sel.size === lib.items.length}
                    ref={(el) => el && (el.indeterminate = multi && lib.sel.size < lib.items.length)}
                    onChange={() => (lib.sel.size === lib.items.length ? lib.clearSel() : lib.selectAll())}
                  />
                )}
                <h1>{route === 'trash' ? '回收站' : '全部图片'}</h1>
                <span className="subtle num">
                  {lib.items.length < lib.base.length ? `${lib.items.length} / ${lib.base.length}` : lib.base.length} 张
                </span>
                {route === 'trash' && lib.base.length > 0 && !multi && (
                  <button type="button" className="btn btn-ghost btn-sm" style={{ color: 'var(--danger)' }} onClick={() => lib.act.purge(lib.base.map((i) => i.id))}>
                    清空回收站
                  </button>
                )}
                <span className="grow" />
                {narrow && (
                  <button type="button" className={`btn btn-secondary filter-btn ${n ? 'active' : ''}`} onClick={openFilters}>
                    <Icon name="filter" />
                    筛选
                    {n ? <span className="filter-n">{n}</span> : null}
                  </button>
                )}
                {!narrow && <SortMenu lib={lib} iconOnly />}
                <ViewToggle lib={lib} />
              </div>
              {(multi || lib.selectMode) && !wideB && <BulkBar lib={lib} />}
              <FilterChips lib={lib} />
              <ImageCollection lib={lib} tile="160px" compact />
              {route === 'trash' && lib.base.length > 0 && <p className="hint" style={{ marginTop: 8 }}>回收站中的图片保留 {lib.settings.trashRetentionDays} 天后自动永久删除。</p>}
            </>
          )}
        </main>
        {showInspector && (
          <aside className="inspector" aria-label="图片详情">
            {multi ? (
              <BatchPanel lib={lib} />
            ) : active ? (
              <DetailPanel lib={lib} image={active} onClose={closeDetail} />
            ) : (
              <div className="inspector-empty">
                <div>
                  <Icon name="image" size={28} />
                  <p style={{ marginTop: 8 }}>选择图片查看链接和详情</p>
                  <p className="hint" style={{ marginTop: 4 }}>
                    按住 Shift 或 Ctrl 可选择多张
                  </p>
                </div>
              </div>
            )}
          </aside>
        )}
      </div>
      {active && !wideB && !page && (
        <Drawer onClose={closeDetail} width="420px" label="图片详情">
          <DetailPanel lib={lib} image={active} onClose={closeDetail} />
        </Drawer>
      )}
      {navOpen && (
        <Drawer side="left" width="272px" onClose={() => setNavOpen(false)} label="导航">
          <div className="sidenav in-drawer">
            <Brand />
            <nav>
              <NavList route={route} go={go} images={images} status={status} />
            </nav>
            <div className="drawer-theme">
              <ThemeButton theme={theme} setTheme={setTheme} />
            </div>
          </div>
        </Drawer>
      )}
      {!trayHidden && <UploadTray lib={lib} onClose={() => (setTrayHidden(true), lib.clearFinished())} />}
    </div>
  );
}

// ================================================================ C — top tabs, upload first

function ShellC({ lib, route, go, theme, setTheme, toast, status, images, page, searchRef, openFilters, active, closeDetail, isMobile }) {
  const tabs = (cls) => (
    <nav className={cls} aria-label="主导航">
      {NAV.map((n) => (
        <button key={n.key} type="button" aria-current={route === n.key ? 'page' : undefined} onClick={() => go(n.key)}>
          {n.short}
          {n.key === 'trash' && navCount(images, 'trash') > 0 && <span className="tab-count">{navCount(images, 'trash')}</span>}
          {n.key === 'status' && status.warnings.length > 0 && <span className="dot tab-dot" aria-label="有警告" />}
        </button>
      ))}
    </nav>
  );
  const groupDays = route === 'images' && lib.view === 'grid' && (lib.sort === 'new' || lib.sort === 'old');
  return (
    <div className="shell-c" data-screen-label="direction-c">
      <header className="header-c">
        <div className="bar">
          <Brand />
          {tabs('tabs')}
          <span className="grow" />
          <ThemeButton theme={theme} setTheme={setTheme} />
          <AccountMenu toast={toast} />
        </div>
        {tabs('tabs tabs-row')}
      </header>
      <main>
        {page ?? (
          <div className="page">
            {route === 'images' && <QuickUpload lib={lib} />}
            <LibraryHead lib={lib} route={route} compact />
            <LibraryToolbar lib={lib} searchRef={searchRef} onOpenFilters={openFilters} />
            <ImageCollection lib={lib} groupDays={groupDays} />
          </div>
        )}
      </main>
      {route === 'images' && !page && !lib.sel.size && (
        <div className="mobile-bar">
          <button type="button" className="btn btn-primary" onClick={lib.pick}>
            <Icon name="upload" />
            上传图片
          </button>
        </div>
      )}
      {active && (
        <Modal title={null} onClose={closeDetail} width="1040px" className="full-sm detail-modal">
          <DetailPanel lib={lib} image={active} onClose={closeDetail} wide />
        </Modal>
      )}
    </div>
  );
}

// ---------------------------------------------------------------- prototype options (review chrome, not product UI)

function ProtoPanel({ dir, setDir, theme, setTheme, ext, setExt, dataState, setDataState, simulate }) {
  const [open, setOpen] = React.useState(false);
  const opt = (cur, key, label, set) => (
    <button key={key} type="button" aria-pressed={cur === key} onClick={() => set(key)}>
      {label}
    </button>
  );
  return (
    <div data-proto="controls">
      <button type="button" className="proto-toggle" aria-expanded={open} onClick={() => setOpen(!open)}>
        原型选项
      </button>
      {open && (
        <div className="proto-panel">
          <label>方案</label>
          <div className="opts">
            {opt(dir, 'a', 'A 侧栏', setDir)}
            {opt(dir, 'b', 'B 三栏', setDir)}
            {opt(dir, 'c', 'C 上传优先', setDir)}
          </div>
          <label>主题</label>
          <div className="opts">
            {opt(theme, 'light', '浅色', setTheme)}
            {opt(theme, 'dark', '深色', setTheme)}
          </div>
          <label>扩展功能（标签 / 重命名 / 时间与来源筛选，需后端支持）</label>
          <div className="opts">
            {opt(ext, true, '显示', setExt)}
            {opt(ext, false, '隐藏', setExt)}
          </div>
          <label>数据状态</label>
          <div className="opts">
            {opt(dataState, 'ok', '正常', setDataState)}
            {opt(dataState, 'empty', '空', setDataState)}
            {opt(dataState, 'loading', '加载中', setDataState)}
            {opt(dataState, 'error', '出错', setDataState)}
          </div>
          <label>上传</label>
          <div className="opts">
            <button type="button" onClick={simulate}>
              模拟上传 5 个文件
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<App />);
