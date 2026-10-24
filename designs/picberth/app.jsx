// App state, the console shell and the prototype options panel.

const PARAMS = new URLSearchParams(location.search);
const EMBED = PARAMS.has('embed');

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
  const [theme, setTheme] = React.useState(PARAMS.get('theme') || 'system');
  const [dataState, setDataStateRaw] = React.useState(PARAMS.get('state') || 'ok');
  const [route, setRoute] = React.useState(PARAMS.get('r') || 'images');
  const [images, setImages] = React.useState(() => (PARAMS.get('state') === 'empty' ? [] : initialImages()));
  const [tokens, setTokens] = React.useState(TOKENS);
  const [settings, setSettings] = React.useState(SETTINGS);
  const [filters, setFilters] = React.useState(EMPTY_FILTERS);
  const [sort, setSort] = React.useState('new');
  const [view, setView] = React.useState(PARAMS.get('view') || 'table');
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
  const [moreOpen, setMoreOpen] = React.useState(false);
  const [adjustOpen, setAdjustOpen] = React.useState(false);
  const [searchOpen, setSearchOpen] = React.useState(false);

  const isMobile = useMedia('(max-width: 720px)');
  const rail = useMedia('(max-width: 1099px)') && !isMobile;
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
  const items = React.useMemo(() => applyFilters(base, filters, sort, true), [base, filters, sort]);
  const knownTags = React.useMemo(() => allTags(images), [images]);
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
    const from = anchor.current;
    setSel((prev) => {
      const next = new Set(prev);
      if (shift && from) {
        const ids = items.map((i) => i.id);
        const a = ids.indexOf(from);
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
  const open = (id) => setActiveId(id);
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
      else if (m.size > settings.maxUploadMB * 1048576) error = `超过 ${settings.maxUploadMB} MB 上限`;
      const dupOf = m.dup ? images.find((i) => i.filename === m.name && i.state === 'active')?.id : null;
      return { ...m, id: Math.random().toString(36).slice(2), format: m.format || 'png', vis: uploadVis, status: error ? 'error' : 'queued', error, retryable: false, progress: 0, dupOf };
    });
    setUploads((u) => [...list, ...u]);
    setUploadOpen(true);
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
      if (!uploadOpen) toast(failed ? `已上传 ${done} 张，失败 ${failed} 张` : `已上传 ${done} 张图片`);
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
  const addUploadsRef = React.useRef(addUploads);
  addUploadsRef.current = addUploads;
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

  const focusSearch = () => {
    setSearchOpen(true);
    setTimeout(() => searchRef.current?.focus(), 0);
  };
  const keyRef = React.useRef(null);
  keyRef.current = (e) => {
    if (isTyping(e) || e.metaKey || e.ctrlKey || e.altKey) return;
    if (document.querySelector('.modal, .drawer, .sheet') && e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return;
    if (e.key === '/' && inLibrary) {
      e.preventDefault();
      focusSearch();
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
    setSearchOpen(false);
    window.scrollTo(0, 0);
    document.querySelector('.page-body')?.scrollTo(0, 0);
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
    if (PARAMS.get('filters')) setAdjustOpen(true);
    if (PARAMS.get('purge')) setPurgeIds(base.slice(0, 1).map((i) => i.id));
    if (PARAMS.get('more')) setMoreOpen(true);
    if (PARAMS.get('f')) setFilters({ ...EMPTY_FILTERS, formats: ['png'], vis: 'public', tags: ['博客'] });
  }, []);

  const lib = {
    images, base, items, byId, route, filters, setFilters, sort, setSort, view, setView, sel, selectMode, setSelectMode,
    activeId, open, clickItem, toggle, neighbors, selectAll: () => setSel(new Set(items.map((i) => i.id))), clearSel,
    suppressClick: () => (suppress.current = true), linkFormat, setLinkFormat, act, dataState,
    retry: () => setDataState('ok'), settings, isMobile, rail, toast, pick: pickFiles, go, knownTags,
    searchRef, searchOpen, setSearchOpen,
    uploads, uploadVis, setUploadVis, cancelUpload, retryUpload, removeUpload, clearFinished,
  };

  const page =
    route === 'tokens' ? (
      <TokensPage tokens={tokens} setTokens={setTokens} toast={toast} />
    ) : route === 'settings' ? (
      <SettingsPage settings={settings} setSettings={setSettings} toast={toast} />
    ) : route === 'status' ? (
      <StatusPage status={STATUS} images={images} toast={toast} />
    ) : null;

  return (
    <>
      <Shell
        lib={lib}
        route={route}
        go={go}
        theme={theme}
        setTheme={setTheme}
        toast={toast}
        status={STATUS}
        page={page}
        moreOpen={moreOpen}
        setMoreOpen={setMoreOpen}
        openAdjust={() => setAdjustOpen(true)}
        setUploadOpen={setUploadOpen}
        uploadOpen={uploadOpen}
        active={active}
        closeDetail={() => setActiveId(null)}
      />
      {adjustOpen && <AdjustSheet lib={lib} onClose={() => setAdjustOpen(false)} />}
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
      {!EMBED && <ProtoPanel theme={theme} setTheme={setTheme} dataState={dataState} setDataState={setDataState} simulate={simulateUploads} />}
    </>
  );
}

// ---------------------------------------------------------------- shell

const DOCK_MS = 250;

function Shell({ lib, route, go, theme, setTheme, toast, status, page, moreOpen, setMoreOpen, openAdjust, uploadOpen, setUploadOpen, active, closeDetail }) {
  const mobile = lib.isMobile;
  const rail = lib.rail;
  const wide = useMedia('(min-width: 1280px)');
  const [measure, width] = useWidth();
  const inLib = !page;
  const canDock = wide && inLib;
  const bulk = lib.sel.size > 0 || lib.selectMode;

  // Keep the docked panel's content while the column collapses.
  const [dockImg, setDockImg] = React.useState(null);
  React.useEffect(() => {
    if (active) return setDockImg(active);
    const t = setTimeout(() => setDockImg(null), DOCK_MS);
    return () => clearTimeout(t);
  }, [active]);

  return (
    <div className={`app ${rail ? 'rail' : ''} ${mobile ? 'mobile' : ''}`} data-screen-label="console">
      {!mobile && <Sidebar route={route} go={go} status={status} theme={theme} setTheme={setTheme} toast={toast} />}
      <div className={`panel ${canDock && active ? 'docked' : ''}`}>
        <main className="panel-main">
          {page ?? (
            <div className="page" ref={measure}>
              <LibraryHead lib={lib} width={width} onUpload={() => setUploadOpen(true)} onAdjust={openAdjust} />
              <FilterChips lib={lib} />
              <div className="page-body">
                <ImageCollection lib={lib} />
              </div>
            </div>
          )}
        </main>
        {canDock && (
          <aside className="dock" aria-label="图片详情" aria-hidden={!active}>
            <div className="dock-inner">{(active || dockImg) && <DetailPanel lib={lib} image={active ?? dockImg} onClose={closeDetail} mode="dock" />}</div>
          </aside>
        )}
      </div>

      {mobile && bulk && inLib && <BulkBar lib={lib} compact floating />}
      {mobile && <TabBar route={route} go={go} status={status} moreOpen={moreOpen} onMore={() => setMoreOpen(true)} />}
      {mobile && moreOpen && <MoreSheet route={route} go={go} status={status} theme={theme} setTheme={setTheme} toast={toast} onClose={() => setMoreOpen(false)} />}

      {active && !canDock && !mobile && (
        <Drawer onClose={closeDetail} width="420px" label="图片详情">
          {(close) => <DetailPanel lib={lib} image={active} onClose={close} mode="drawer" />}
        </Drawer>
      )}
      {active && mobile && (
        <Sheet onClose={closeDetail} label="图片详情" full>
          {(close) => <DetailPanel lib={lib} image={active} onClose={close} mode="sheet" />}
        </Sheet>
      )}
      {uploadOpen && <UploadDialog lib={lib} onClose={() => (setUploadOpen(false), lib.uploads.some(isActiveUpload) || lib.clearFinished())} />}
    </div>
  );
}

// ---------------------------------------------------------------- prototype options (review chrome, not product UI)

function ProtoPanel({ theme, setTheme, dataState, setDataState, simulate }) {
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
          <label>主题</label>
          <div className="opts">
            {THEMES.map((t) => opt(theme, t.key, t.label, setTheme))}
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
