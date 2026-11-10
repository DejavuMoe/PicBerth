// Console shell: sidebar with the folder tree, routing, command palette, uploads, leave guard.

const NAV = [
  { group: 0, key: 'library', label: '图片', icon: 'image' },
  { group: 0, key: 'trash', label: '回收站', icon: 'trash' },
  { group: 1, key: 'tokens', label: '令牌', icon: 'key' },
  { group: 1, key: 'settings', label: '设置', icon: 'sliders' },
  { group: 1, key: 'status', label: '状态', icon: 'pulse' },
];

// Routes mirror the production paths: #/library, #/library/<folder names>, #/trash, #/tokens, #/settings, #/status.
// An empty hash is `/`: signed-in users go to the library.
function readLoc() {
  const raw = location.hash.replace(/^#\/?/, '').split('?')[0];
  const segs = raw.split('/').filter(Boolean).map((s) => {
    try {
      return decodeURIComponent(s);
    } catch (e) {
      return s;
    }
  });
  return { raw, route: segs[0] || 'library', rest: segs.slice(1) };
}

function useUploads(setImages, settings) {
  const [items, setItems] = useState([]);
  const timers = useRef({});
  const step = (id) => {
    setItems((xs) =>
      xs.map((u) => {
        if (u.id !== id) return u;
        if (u.status === 'queued') return { ...u, status: 'uploading', progress: 4 };
        if (u.status === 'uploading') {
          const p = Math.min(100, u.progress + 9 + Math.random() * 14);
          if (u.failAt && p >= u.failAt) {
            clearInterval(timers.current[id]);
            return { ...u, status: 'error', error: '网络连接中断', retryable: true, failAt: null };
          }
          return p >= 100 ? { ...u, status: 'processing', progress: 100 } : { ...u, progress: p };
        }
        if (u.status === 'processing') {
          clearInterval(timers.current[id]);
          if (u.duplicateOf) return { ...u, status: 'duplicate', imageId: u.duplicateOf };
          const im = {
            ...initialImages()[0],
            id: fakeId(Math.floor(Math.random() * 1e6)),
            filename: u.name,
            thumb: u.thumb,
            size: u.size,
            source: 'web',
            folderId: u.folderId,
            createdAt: NOW,
          };
          im.links = makeLinks(im.id, im.ext, im.filename);
          setImages((xs) => [im, ...xs]);
          return { ...u, status: 'done', imageId: im.id };
        }
        return u;
      }),
    );
  };
  const start = (id) => {
    clearInterval(timers.current[id]);
    timers.current[id] = setInterval(() => step(id), 260);
  };
  const add = (files, folderId = null) => {
    const next = files.map((f) => {
      const id = Math.random().toString(36).slice(2);
      const base = { id, name: f.name, size: f.size, folderId, thumb: f.thumb || (f instanceof Blob ? URL.createObjectURL(f) : ''), progress: 0, failAt: f.failAt, duplicateOf: f.duplicateOf };
      if (f.type && !/^image\/(jpeg|png|gif|webp|avif)$/.test(f.type)) return { ...base, status: 'error', error: '不支持的格式', retryable: false };
      if (f.size > settings.maxUploadMB * 1048576) return { ...base, status: 'error', error: `超过单文件上限 ${settings.maxUploadMB} MB`, retryable: false };
      return { ...base, status: 'queued' };
    });
    setItems((xs) => [...next, ...xs]);
    next.filter((u) => u.status === 'queued').forEach((u, i) => setTimeout(() => start(u.id), i * 300));
  };
  return {
    items,
    add,
    cancel: (id) => (clearInterval(timers.current[id]), setItems((xs) => xs.filter((u) => u.id !== id))),
    retry: (id) => (setItems((xs) => xs.map((u) => (u.id === id ? { ...u, status: 'queued', error: null, progress: 0 } : u))), start(id)),
    remove: (id) => setItems((xs) => xs.filter((u) => u.id !== id)),
    clear: () => setItems((xs) => xs.filter((u) => u.status !== 'done' && u.status !== 'duplicate' && u.status !== 'error')),
  };
}

// ---------------------------------------------------------------- uploads panel
/** Uploads float over the corner of the window so the image list never jumps. */
function UploadDock({ items, folders, onCancel, onRetry, onRemove, onCopy, onCopyAll, onClear, onOpen }) {
  const [folded, setFolded] = useState(false);
  useEffect(() => {
    if (items.some((u) => u.status === 'queued')) setFolded(false);
  }, [items.length]);
  if (!items.length) return null;
  const done = items.filter((u) => u.status === 'done' || u.status === 'duplicate');
  const failed = items.filter((u) => u.status === 'error');
  const busy = items.some((u) => u.status === 'queued' || u.status === 'uploading' || u.status === 'processing');
  return (
    <section className="dock" aria-label="上传">
      <div className="dock-head">
        <span className="t">{busy ? '正在上传' : done.length ? '上传完成' : '上传失败'}</span>
        <span className="muted num">
          {done.length} / {items.length}
        </span>
        <span className="grow" />
        {done.length > 1 && (
          <button type="button" className="btn btn-ghost btn-sm" onClick={onCopyAll}>
            <Icon name="copy" size="s" />
            复制全部链接
          </button>
        )}
        <button type="button" className="btn btn-ghost btn-sm btn-icon" aria-label={folded ? '展开' : '收起'} title={folded ? '展开' : '收起'} aria-expanded={!folded} onClick={() => setFolded((f) => !f)}>
          <Icon name={folded ? 'up' : 'down'} />
        </button>
        {!busy && (
          <button type="button" className="btn btn-ghost btn-sm btn-icon" aria-label="关闭" title="关闭" onClick={onClear}>
            <Icon name="x" />
          </button>
        )}
      </div>
      {!folded && (
        <div className="dock-list">
          {items.map((u) => (
            <div key={u.id} className="up-row">
              <img className="thumb" src={u.thumb} alt="" />
              <div className="up-name">
                <span className="ellipsis medium">{u.name}</span>
                {(u.status === 'uploading' || u.status === 'queued') && (
                  <div className="bar" role="progressbar" aria-label={u.name} aria-valuenow={Math.round(u.progress)} aria-valuemin={0} aria-valuemax={100}>
                    <i style={{ width: `${u.progress}%` }} />
                  </div>
                )}
                {u.status === 'processing' && (
                  <div className="bar indeterminate" role="progressbar" aria-label={u.name}>
                    <i />
                  </div>
                )}
                {u.status === 'error' && <span className="s text-danger">{u.error}</span>}
                {u.status === 'done' && (
                  <span className="s muted facts">
                    <span>{fmtBytes(u.size)}</span>
                    {u.folderId && <span className="ellipsis">{folderLabel(folders, u.folderId)}</span>}
                  </span>
                )}
                {u.status === 'duplicate' && <span className="s muted">已存在，使用原有链接</span>}
              </div>
              <div className="row" style={{ gap: 2 }}>
                {u.status === 'queued' && <span className="s muted">等待中</span>}
                {u.status === 'uploading' && <span className="s muted num">{Math.round(u.progress)}%</span>}
                {u.status === 'processing' && <span className="s muted">处理中</span>}
                {(u.status === 'queued' || u.status === 'uploading') && (
                  <button type="button" className="btn btn-ghost btn-sm btn-icon" aria-label={`取消 ${u.name}`} title="取消" onClick={() => onCancel(u.id)}>
                    <Icon name="x" />
                  </button>
                )}
                {u.status === 'error' && u.retryable && (
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => onRetry(u.id)}>
                    重试
                  </button>
                )}
                {u.status === 'error' && (
                  <button type="button" className="btn btn-ghost btn-sm btn-icon" aria-label={`移除 ${u.name}`} title="移除" onClick={() => onRemove(u.id)}>
                    <Icon name="x" />
                  </button>
                )}
                {(u.status === 'done' || u.status === 'duplicate') && (
                  <>
                    <button type="button" className="btn btn-ghost btn-sm" onClick={() => onOpen(u.imageId)}>
                      查看
                    </button>
                    <button type="button" className="btn btn-secondary btn-sm" onClick={() => onCopy(u.imageId)}>
                      <Icon name="copy" size="s" />
                      复制
                    </button>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
      <span className="sr-only" role="status">
        {!busy && (failed.length ? `${done.length} 张已上传，${failed.length} 张失败` : `${done.length} 张已上传`)}
      </span>
    </section>
  );
}

// ---------------------------------------------------------------- command palette
function Palette({ onClose, images, folders, commands, onOpenImage, onCopy, onOpenFolder }) {
  const [q, setQ] = useState('');
  const [idx, setIdx] = useState(0);
  const listRef = useRef(null);
  const groups = useMemo(() => {
    const s = q.trim().toLowerCase();
    const out = [];
    if (s) {
      const ims = images.filter((im) => im.filename.toLowerCase().includes(s) || im.id.startsWith(s) || im.sha256.startsWith(s)).slice(0, 6);
      if (ims.length) out.push({ label: '图片', items: ims.map((im) => ({ key: 'i' + im.id, image: im, title: im.filename, sub: folderLabel(folders, im.folderId), run: () => onOpenImage(im.id) })) });
      const fs = folders.filter((f) => f.name.toLowerCase().includes(s)).slice(0, 5);
      if (fs.length) out.push({ label: '文件夹', items: fs.map((f) => ({ key: 'f' + f.id, icon: 'folder', title: f.name, sub: folderLabel(folders, f.parentId ?? null), run: () => onOpenFolder(f.id) })) });
    }
    for (const g of commands) {
      const items = g.items.filter((c) => !s || c.title.toLowerCase().includes(s));
      if (items.length) out.push({ label: g.label, items });
    }
    return out;
  }, [q, images, folders, commands]);
  const flat = groups.flatMap((g) => g.items);
  const cur = flat[Math.min(idx, flat.length - 1)];
  useEffect(() => setIdx(0), [q]);
  useEffect(() => {
    listRef.current?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' });
  }, [idx, q]);
  const run = (it, alt) => {
    onClose();
    if (alt && it.image) onCopy(it.image);
    else it.run();
  };
  const onKey = (e) => {
    if (e.key === 'ArrowDown') (e.preventDefault(), setIdx((i) => Math.min(flat.length - 1, i + 1)));
    else if (e.key === 'ArrowUp') (e.preventDefault(), setIdx((i) => Math.max(0, i - 1)));
    else if (e.key === 'Enter' && cur) (e.preventDefault(), run(cur, e.metaKey || e.ctrlKey));
    else if (e.key === 'Escape') (e.preventDefault(), e.stopPropagation(), onClose());
  };
  return ReactDOM.createPortal(
    <div className="cmdk-scrim" onPointerDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="cmdk" role="dialog" aria-modal="true" aria-label="命令面板" onKeyDown={onKey}>
        <div className="cmdk-input">
          <Icon name="search" />
          <input autoFocus value={q} placeholder="搜索图片、文件夹或操作" aria-label="搜索图片、文件夹或操作" role="combobox" aria-expanded="true" aria-controls="cmdk-list" aria-activedescendant={cur ? 'cmdk-' + cur.key : undefined} onChange={(e) => setQ(e.target.value)} />
        </div>
        <div className="cmdk-list" id="cmdk-list" role="listbox" ref={listRef}>
          {!flat.length && <div className="cmdk-empty">没有匹配的结果</div>}
          {groups.map((g) => (
            <div key={g.label} role="group" aria-label={g.label}>
              <div className="cmdk-group">{g.label}</div>
              {g.items.map((it) => (
                <button
                  key={it.key}
                  id={'cmdk-' + it.key}
                  type="button"
                  tabIndex={-1}
                  className="cmdk-item"
                  role="option"
                  aria-selected={it === cur}
                  onPointerMove={() => it !== cur && setIdx(flat.indexOf(it))}
                  onClick={(e) => run(it, e.metaKey || e.ctrlKey)}
                >
                  {it.image ? <Thumb img={it.image} /> : <Icon name={it.icon} />}
                  <span className="t ellipsis">{it.title}</span>
                  {it.sub && <span className="sub">{it.sub}</span>}
                  {it.kbd && <kbd>{it.kbd}</kbd>}
                  {it.image && it === cur && (
                    <span className="sub" style={{ marginLeft: 0 }}>
                      {MOD} ↵ 复制链接
                    </span>
                  )}
                </button>
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>,
    document.body,
  );
}

// ---------------------------------------------------------------- app
function App() {
  const toast = useToast();
  const [themePref, setThemePref] = useTheme();
  const [loc, setLoc] = useState(readLoc);
  const route = loc.route;
  const [images, setImages] = useState(initialImages);
  const [folders, setFolders] = useState(FOLDERS);
  const [folderState, setFolderState] = useState(() => new URLSearchParams(location.search).get('folders') || 'ready');
  const [settings, setSettings] = useState(SETTINGS);
  const [dataState, setDataState] = useState(() => new URLSearchParams(location.search).get('data') || 'ready');
  const [leave, setLeave] = useState(null);
  const [dragging, setDragging] = useState(false);
  const [palette, setPalette] = useState(false);
  const [linkFormat, setLinkFormat] = useState('url');
  const [openRequest, setOpenRequest] = useState(null);
  const [createSignal, setCreateSignal] = useState(0);
  const [tokenSignal, setTokenSignal] = useState(0);
  const guard = useRef(false);
  const lib = useRef({});
  const fileRef = useRef(null);
  const uploads = useUploads(setImages, settings);
  const phone = useMedia('(max-width: 720px)');
  const wideNav = useMedia('(min-width: 1100px)');
  const initialOpen = useMemo(() => new URLSearchParams(location.search).get('open'), []);

  useEffect(() => {
    if (!loc.raw) {
      history.replaceState(null, '', '#/library');
      setLoc(readLoc());
    }
  }, []);
  useEffect(() => {
    const on = () => {
      const l = readLoc();
      if (!l.raw) {
        history.replaceState(null, '', '#/library');
        setLoc(readLoc());
        return;
      }
      if (guard.current && route === 'settings' && l.route !== 'settings') {
        history.replaceState(null, '', '#/settings');
        setLeave(l.raw);
        return;
      }
      setLoc(l);
    };
    addEventListener('hashchange', on);
    return () => removeEventListener('hashchange', on);
  }, [route]);
  const go = (r) => {
    if (r === (loc.raw || 'library')) return;
    if (guard.current && route === 'settings') return setLeave(r);
    location.hash = '#/' + r;
  };
  // Folder renames and moves keep the address in step with the folder being viewed.
  const navigateHash = (hash, opts = {}) => {
    if (opts.replace) {
      history.replaceState(null, '', hash);
      setLoc(readLoc());
    } else location.hash = hash;
  };
  const folderList = folderState === 'none' ? [] : folders;
  const treeState = folderState === 'none' ? 'ready' : folderState;
  const resolved = route === 'library' && treeState === 'ready' ? resolvePath(folderList, loc.rest) : null;
  const missing = resolved === undefined;
  const folderId = resolved ?? null;
  const shownImages = dataState === 'empty' ? [] : images;
  const active = useMemo(() => shownImages.filter((im) => im.state === 'active'), [shownImages]);
  const counts = useMemo(() => {
    const m = new Map();
    for (const im of active) m.set(im.folderId ?? null, (m.get(im.folderId ?? null) || 0) + 1);
    return m;
  }, [active]);

  // ---- links
  const fmtName = (k) => LINK_FORMATS.find((f) => f.key === k)?.label;
  const copyLink = (im, fmt = linkFormat) => {
    if (!im) return;
    if (fmt === 'id' || fmt === 'sha256') {
      copyText(im[fmt]);
      toast(fmt === 'id' ? '已复制 ID' : '已复制 SHA-256');
      return;
    }
    setLinkFormat(fmt);
    copyText(im.links[fmt]);
    toast(`已复制 ${fmtName(fmt)}`);
  };
  const copyMany = (ims) => {
    copyText(ims.map((im) => im.links[linkFormat]).join('\n'));
    toast(`已复制 ${ims.length} 条 ${fmtName(linkFormat)}`);
  };

  // ---- folders
  const openFolder = (id) => navigateHash(libraryHash(folders, id));
  const openImage = (id) => {
    const im = images.find((x) => x.id === id);
    if (!im) return;
    const hash = libraryHash(folders, im.folderId ?? null);
    const moved = location.hash !== hash;
    if (moved) location.hash = hash;
    setTimeout(() => setOpenRequest({ id }), moved ? 80 : 0);
  };
  const moveImages = (ids, to) => {
    const moving = images.filter((im) => ids.includes(im.id) && im.state === 'active' && (im.folderId ?? null) !== to).map((im) => im.id);
    if (!moving.length) return;
    const before = images;
    lib.current.leaving?.(moving);
    setImages((xs) => xs.map((im) => (moving.includes(im.id) ? { ...im, folderId: to } : im)));
    const where = folderLabel(folders, to);
    toast(moving.length > 1 ? `已将 ${moving.length} 张图片移到 ${where}` : `已移到 ${where}`, { label: '撤销', run: () => setImages(before) });
  };
  const inside = (id) => route === 'library' && folderId !== null && (folderId === id || descendantIds(folders, id).has(folderId));
  const createFolder = (parentId, name) => {
    const e = folderNameError(folders, parentId, name);
    if (e) return e;
    if (folderDepth(folders, parentId) >= MAX_DEPTH) return `最多 ${MAX_DEPTH} 层`;
    setFolders((xs) => [...xs, { id: 'f' + Math.random().toString(36).slice(2, 8), name: name.trim(), parentId }]);
    toast('已创建文件夹');
    return '';
  };
  const renameFolder = (id, name) => {
    const f = folderById(folders, id);
    const e = folderNameError(folders, f.parentId ?? null, name, id);
    if (e) return e;
    const next = folders.map((x) => (x.id === id ? { ...x, name: name.trim() } : x));
    setFolders(next);
    if (inside(id)) navigateHash(libraryHash(next, folderId), { replace: true });
    toast('已重命名');
    return '';
  };
  const moveFolder = (id, to) => {
    const f = folderById(folders, id);
    if (folderNameError(folders, to, f.name, id)) return toast('目标位置已有同名文件夹');
    if (!fitsUnder(folders, id, to)) return toast(`最多 ${MAX_DEPTH} 层`);
    const before = folders;
    const next = folders.map((x) => (x.id === id ? { ...x, parentId: to } : x));
    const here = inside(id);
    setFolders(next);
    if (here) navigateHash(libraryHash(next, folderId), { replace: true });
    toast(`已将 ${f.name} 移到 ${folderLabel(folders, to)}`, {
      label: '撤销',
      run: () => (setFolders(before), here && navigateHash(libraryHash(before, folderId), { replace: true })),
    });
  };
  const deleteFolder = (id) => {
    const f = folderById(folders, id);
    if (childrenOf(folders, id).length || (counts.get(id) || 0) > 0) return toast('文件夹不为空');
    const beforeFolders = folders;
    setFolders((xs) => xs.filter((x) => x.id !== id));
    if (route === 'library' && folderId === id) navigateHash(libraryHash(folders, f.parentId ?? null), { replace: true });
    toast('已删除文件夹', { label: '撤销', run: () => setFolders(beforeFolders) });
  };
  const panelProps = {
    folders: folderList,
    state: treeState,
    currentId: route === 'library' && !missing ? folderId : undefined,
    counts,
    onOpen: openFolder,
    onCreate: createFolder,
    onRename: renameFolder,
    onMove: moveFolder,
    onDelete: deleteFolder,
    onDropImages: moveImages,
    onRetry: () => setFolderState('ready'),
  };

  // ---- uploads: file picker, window-wide drop and paste
  const pickFiles = () => fileRef.current?.click();
  const uploadTo = (files) => {
    uploads.add(files, folderId);
    if (route !== 'library') location.hash = '#/library';
  };
  useEffect(() => {
    let depth = 0;
    const hasFiles = (e) => [...(e.dataTransfer?.types || [])].includes('Files');
    const enter = (e) => hasFiles(e) && (depth++, setDragging(true));
    const leaveEv = (e) => hasFiles(e) && --depth <= 0 && ((depth = 0), setDragging(false));
    const over = (e) => hasFiles(e) && e.preventDefault();
    const drop = (e) => {
      if (!hasFiles(e)) return;
      e.preventDefault();
      depth = 0;
      setDragging(false);
      uploadTo([...e.dataTransfer.files]);
    };
    const paste = (e) => {
      if (/INPUT|TEXTAREA/.test(document.activeElement?.tagName)) return;
      const files = [...(e.clipboardData?.files || [])];
      if (files.length) uploadTo(files);
    };
    const key = (e) => {
      if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K')) {
        e.preventDefault();
        setPalette((p) => !p);
        return;
      }
      const typing = /INPUT|TEXTAREA/.test(document.activeElement?.tagName) || document.querySelector('.scrim, .cmdk-scrim, .viewer');
      if (typing || e.metaKey || e.ctrlKey || e.altKey) return;
      if ((e.key === 'u' || e.key === 'U') && route === 'library') pickFiles();
    };
    addEventListener('dragenter', enter);
    addEventListener('dragleave', leaveEv);
    addEventListener('dragover', over);
    addEventListener('drop', drop);
    addEventListener('paste', paste);
    addEventListener('keydown', key);
    return () => {
      removeEventListener('dragenter', enter);
      removeEventListener('dragleave', leaveEv);
      removeEventListener('dragover', over);
      removeEventListener('drop', drop);
      removeEventListener('paste', paste);
      removeEventListener('keydown', key);
    };
  }, [route, folderId]);

  const simulateUploads = () =>
    uploadTo([
      { name: 'release-notes-cover.png', size: 812345, thumb: 'assets/fixtures/banner-wide.webp', type: 'image/png' },
      { name: 'IMG_5012.jpg', size: 2400123, thumb: 'assets/fixtures/desert-noon.jpg', type: 'image/jpeg', failAt: 60 },
      { name: 'handler-retry.png', size: 650000, thumb: 'assets/fixtures/code-snippet.png', type: 'image/png', duplicateOf: images[1]?.id },
      { name: 'scan-raw.tiff', size: 31000000, thumb: 'assets/fixtures/receipt-scan.jpg', type: 'image/tiff' },
    ]);

  const warnings = STATUS.warnings.length;
  const trashCount = shownImages.filter((im) => im.state === 'trashed').length;

  const commands = useMemo(
    () => [
      {
        label: '操作',
        items: [
          { key: 'upload', icon: 'upload', title: '上传图片', kbd: 'U', run: () => (route !== 'library' && go('library'), pickFiles()) },
          ...(wideNav ? [{ key: 'folder', icon: 'folder-plus', title: '新建文件夹', run: () => setCreateSignal((n) => n + 1) }] : []),
          { key: 'token', icon: 'key', title: '新建令牌', run: () => (go('tokens'), setTimeout(() => setTokenSignal((n) => n + 1), 80)) },
        ],
      },
      {
        label: '前往',
        items: [
          ...NAV.map((n) => ({ key: 'go-' + n.key, icon: n.icon, title: n.label, run: () => go(n.key) })),
          { key: 'go-docs', icon: 'book', title: 'API 文档', run: () => window.open('api-docs.html', '_blank', 'noopener') },
        ],
      },
      {
        label: '主题',
        items: THEMES.map((t) => ({ key: 'theme-' + t.value, icon: t.icon, title: t.label, sub: themePref === t.value ? '当前' : '', run: () => setThemePref(t.value) })),
      },
    ],
    [route, loc.raw, wideNav, themePref],
  );

  let page;
  if (route === 'library' || route === 'trash')
    page = (
      <Library
        mode={route === 'trash' ? 'trash' : 'active'}
        images={shownImages}
        setImages={setImages}
        folders={folderList}
        folderState={treeState}
        folderId={folderId}
        missing={missing}
        panelProps={panelProps}
        onOpenFolder={openFolder}
        moveImages={moveImages}
        onUpload={pickFiles}
        settings={settings}
        dataState={dataState === 'empty' ? 'ready' : dataState}
        onRetryLoad={() => setDataState('ready')}
        onRetryFolders={() => setFolderState('ready')}
        initialOpen={route === 'library' && !phone ? initialOpen : null}
        openRequest={openRequest}
        hook={lib}
        copyLink={copyLink}
        copyMany={copyMany}
      />
    );
  else if (route === 'tokens') page = <Tokens dataState={dataState} onRetryLoad={() => setDataState('ready')} createSignal={tokenSignal} />;
  else if (route === 'settings') page = <Settings settings={settings} setSettings={setSettings} guard={guard} />;
  else if (route === 'status') page = <Status dataState={dataState} onRetryLoad={() => setDataState('ready')} settings={settings} />;
  else
    page = (
      <div className="panel-main">
        <div className="nf">
          <span className="nf-title">页面不存在</span>
          <span className="muted mono">{location.hash.slice(1)}</span>
          <button type="button" className="btn btn-secondary" onClick={() => go('library')}>
            返回图片
          </button>
        </div>
      </div>
    );

  const navLink = (n) => (
    <a key={n.key} href={'#/' + n.key} className="nav-item" aria-current={route === n.key ? 'page' : undefined} title={n.label} onClick={(e) => (e.preventDefault(), go(n.key))}>
      <Icon name={n.icon} />
      <span className="label">{n.label}</span>
      {n.key === 'trash' && trashCount > 0 && <span className="count">{trashCount}</span>}
      {n.key === 'status' && warnings > 0 && <span className="dot dot-warning" aria-label={`${warnings} 条警告`} />}
    </a>
  );

  return (
    <div className="shell">
      <nav className="side" aria-label="主导航">
        <div className="brand-row">
          <span className="brand">
            <span className="brand-mark" aria-hidden="true" />
            <span className="brand-word">PicBerth</span>
          </span>
          <button type="button" className="cmdk-btn" aria-label="命令面板" title="命令面板" aria-expanded={palette} onClick={() => setPalette(true)}>
            {MOD} K
          </button>
        </div>
        {wideNav ? (
          <div className="side-scroll">
            <FolderPanel {...panelProps} createSignal={createSignal} />
          </div>
        ) : (
          <div className="side-nav">{navLink(NAV[0])}</div>
        )}
        <div className="side-nav">{navLink(NAV[1])}</div>
        <div className="side-sep" />
        <div className="side-nav">{NAV.filter((n) => n.group === 1).map(navLink)}</div>
        <div className="side-fill" />
        <div className="side-nav" style={{ marginTop: 4 }}>
          <Popover
            align="start"
            trigger={(p) => (
              <button {...p} type="button" className="account" aria-label="admin">
                <span className="avatar">A</span>
                <span className="label">admin</span>
              </button>
            )}
          >
            {(close) => (
              <>
                <div className="menu-group">主题</div>
                {THEMES.map((t) => (
                  <MenuItem key={t.value} checked={themePref === t.value} onClick={() => (setThemePref(t.value), close())}>
                    {t.label}
                  </MenuItem>
                ))}
                <div className="menu-sep" />
                <MenuItem icon="logout" onClick={() => (location.href = 'auth.html')}>
                  退出
                </MenuItem>
              </>
            )}
          </Popover>
        </div>
      </nav>

      <main className="main">
        <div className="panel">{page}</div>
      </main>

      <nav className="tabbar" aria-label="主导航">
        {NAV.map((n) => (
          <a key={n.key} href={'#/' + n.key} className="tab" aria-current={route === n.key ? 'page' : undefined} onClick={(e) => (e.preventDefault(), go(n.key))}>
            <Icon name={n.icon} />
            {n.label}
            {n.key === 'status' && warnings > 0 && <span className="dot dot-warning" aria-label={`${warnings} 条警告`} />}
          </a>
        ))}
      </nav>

      <input ref={fileRef} type="file" accept="image/*" multiple hidden onChange={(e) => (uploadTo([...e.target.files]), (e.target.value = ''))} />
      <UploadDock
        items={uploads.items}
        folders={folders}
        onCancel={uploads.cancel}
        onRetry={uploads.retry}
        onRemove={uploads.remove}
        onClear={uploads.clear}
        onOpen={openImage}
        onCopy={(id) => copyLink(images.find((im) => im.id === id))}
        onCopyAll={() => copyMany(uploads.items.filter((u) => u.imageId).map((u) => images.find((im) => im.id === u.imageId)).filter(Boolean))}
      />
      {dragging && (
        <div className="dropzone">
          <DashFrame radius={12} />
          <span className="drop-label">
            <span className="medium">松开以上传</span>
            {route === 'library' && folderId && <span className="where">{folderLabel(folders, folderId)}</span>}
          </span>
        </div>
      )}
      {palette && <Palette onClose={() => setPalette(false)} images={active} folders={folderList} commands={commands} onOpenImage={openImage} onCopy={copyLink} onOpenFolder={openFolder} />}

      {leave && (
        <Dialog
          title="离开设置？"
          onClose={() => setLeave(null)}
          footer={
            <>
              <button type="button" className="btn btn-secondary" data-autofocus onClick={() => setLeave(null)}>
                留下
              </button>
              <button
                type="button"
                className="btn btn-danger"
                onClick={() => {
                  guard.current = false;
                  const r = leave;
                  setLeave(null);
                  location.hash = '#/' + r;
                }}
              >
                放弃更改
              </button>
            </>
          }
        >
          <p>未保存的更改会丢失。</p>
        </Dialog>
      )}

      <ProtoControls
        dataState={dataState}
        setDataState={setDataState}
        folderState={folderState}
        setFolderState={setFolderState}
        themePref={themePref}
        setThemePref={setThemePref}
        onUploads={simulateUploads}
        onDrop={() => (setDragging(true), setTimeout(() => setDragging(false), 2600))}
        onPalette={() => setPalette(true)}
      />
    </div>
  );
}

/** Review-only control for switching prototype states. Not part of the product. */
function ProtoControls({ dataState, setDataState, folderState, setFolderState, themePref, setThemePref, onUploads, onDrop, onPalette }) {
  return (
    <Popover
      align="start"
      trigger={(p) => (
        <button {...p} type="button" className="proto">
          原型
        </button>
      )}
    >
      {(close) => (
        <>
          <div className="menu-group">数据</div>
          {[
            ['ready', '正常'],
            ['empty', '空'],
            ['loading', '加载中'],
            ['error', '加载失败'],
          ].map(([v, l]) => (
            <MenuItem key={v} checked={dataState === v} onClick={() => (setDataState(v), close())}>
              {l}
            </MenuItem>
          ))}
          <div className="menu-sep" />
          <div className="menu-group">文件夹</div>
          {[
            ['ready', '正常'],
            ['none', '没有文件夹'],
            ['loading', '加载中'],
            ['error', '加载失败'],
          ].map(([v, l]) => (
            <MenuItem key={v} checked={folderState === v} onClick={() => (setFolderState(v), close())}>
              {l}
            </MenuItem>
          ))}
          <div className="menu-sep" />
          <div className="menu-group">主题</div>
          {THEMES.map((t) => (
            <MenuItem key={t.value} checked={themePref === t.value} onClick={() => (setThemePref(t.value), close())}>
              {t.label}
            </MenuItem>
          ))}
          <div className="menu-sep" />
          <MenuItem icon="upload" onClick={() => (onUploads(), close())}>
            模拟上传
          </MenuItem>
          <MenuItem icon="download" onClick={() => (onDrop(), close())}>
            拖入文件
          </MenuItem>
          <MenuItem icon="command" onClick={() => (close(), onPalette())}>
            命令面板
          </MenuItem>
          <MenuItem icon="alert" onClick={() => ((location.hash = '#/missing'), close())}>
            不存在的页面
          </MenuItem>
          <MenuItem icon="folder" onClick={() => ((location.hash = '#/library/' + encodeURIComponent('不存在')), close())}>
            不存在的文件夹
          </MenuItem>
          <MenuItem icon="external" onClick={() => (location.href = 'api-docs.html')}>
            API 文档
          </MenuItem>
          <MenuItem icon="logout" onClick={() => (location.href = 'auth.html')}>
            登录与初始化
          </MenuItem>
        </>
      )}
    </Popover>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <ToastHost>
    <App />
  </ToastHost>,
);
