// Console shell: navigation, routing, uploads, leave guard.

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

function App() {
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
  const guard = useRef(false);
  const uploads = useUploads(setImages, settings);
  const phone = useMedia('(max-width: 720px)');
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
  const resolvedFolder = route === 'library' && folderState === 'ready' ? resolvePath(folders, loc.rest) : null;
  const uploadFolder = typeof resolvedFolder === 'string' ? resolvedFolder : null;
  const dropLabel = uploadFolder ? `松开以上传到 ${folderLabel(folders, uploadFolder)}` : '松开以上传';

  // window-wide drop and paste
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
      uploads.add([...e.dataTransfer.files], uploadFolder);
      if (route !== 'library') location.hash = '#/library';
    };
    const paste = (e) => {
      if (/INPUT|TEXTAREA/.test(document.activeElement?.tagName)) return;
      const files = [...(e.clipboardData?.files || [])];
      if (files.length) {
        uploads.add(files, uploadFolder);
        if (route !== 'library') location.hash = '#/library';
      }
    };
    addEventListener('dragenter', enter);
    addEventListener('dragleave', leaveEv);
    addEventListener('dragover', over);
    addEventListener('drop', drop);
    addEventListener('paste', paste);
    return () => {
      removeEventListener('dragenter', enter);
      removeEventListener('dragleave', leaveEv);
      removeEventListener('dragover', over);
      removeEventListener('drop', drop);
      removeEventListener('paste', paste);
    };
  }, [route, uploadFolder]);

  const simulateUploads = () =>
    uploads.add([
      { name: 'release-notes-cover.png', size: 812345, thumb: 'assets/fixtures/banner-wide.webp', type: 'image/png' },
      { name: 'IMG_5012.jpg', size: 2400123, thumb: 'assets/fixtures/desert-noon.jpg', type: 'image/jpeg', failAt: 60 },
      { name: 'handler-retry.png', size: 650000, thumb: 'assets/fixtures/code-snippet.png', type: 'image/png', duplicateOf: images[1]?.id },
      { name: 'scan-raw.tiff', size: 31000000, thumb: 'assets/fixtures/receipt-scan.jpg', type: 'image/tiff' },
    ], uploadFolder);

  const warnings = STATUS.warnings.length;
  const trashCount = images.filter((im) => im.state === 'trashed').length;

  let page;
  if (route === 'library' || route === 'trash')
    page = (
      <Library
        mode={route === 'trash' ? 'trash' : 'active'}
        images={dataState === 'empty' ? [] : images}
        setImages={setImages}
        folders={folderState === 'none' ? [] : folders}
        setFolders={setFolders}
        folderState={folderState === 'none' ? 'ready' : folderState}
        folderPath={route === 'library' ? loc.rest : []}
        onNavigate={navigateHash}
        onRetryFolders={() => setFolderState('ready')}
        uploads={uploads.items}
        uploadApi={uploads}
        settings={settings}
        dataState={dataState === 'empty' ? 'ready' : dataState}
        onRetryLoad={() => setDataState('ready')}
        initialOpen={route === 'library' && !phone ? initialOpen : null}
      />
    );
  else if (route === 'tokens') page = <Tokens dataState={dataState} onRetryLoad={() => setDataState('ready')} />;
  else if (route === 'settings') page = <Settings settings={settings} setSettings={setSettings} guard={guard} />;
  else if (route === 'status') page = <Status dataState={dataState} onRetryLoad={() => setDataState('ready')} settings={settings} />;
  else
    page = (
      <div className="panel-main">
        <div className="nf">
          <span className="code">页面不存在</span>
          <span className="muted mono">{location.hash.slice(1)}</span>
          <button type="button" className="btn btn-secondary" onClick={() => go('library')}>
            返回图片
          </button>
        </div>
      </div>
    );

  return (
    <div className="shell">
      <nav className="side" aria-label="主导航">
        <div className="brand">
          <span className="brand-mark" aria-hidden="true" />
          <span className="brand-word">PicBerth</span>
        </div>
        {[0, 1].map((g) => (
          <div className="nav" key={g}>
            {NAV.filter((n) => n.group === g).map((n) => (
              <a
                key={n.key}
                href={'#/' + n.key}
                className="nav-item"
                aria-current={route === n.key ? 'page' : undefined}
                title={n.label}
                onClick={(e) => (e.preventDefault(), go(n.key))}
              >
                <Icon name={n.icon} />
                <span className="label">{n.label}</span>
                {n.key === 'trash' && trashCount > 0 && <span className="count">{trashCount}</span>}
                {n.key === 'status' && warnings > 0 && <span className="dot dot-warning" aria-label={`${warnings} 条警告`} />}
              </a>
            ))}
          </div>
        ))}
        <div className="side-foot">
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
        <div className="panel">
          {page}
          {dragging && <div className="dropzone">{dropLabel}</div>}
        </div>
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

      <ProtoControls dataState={dataState} setDataState={setDataState} folderState={folderState} setFolderState={setFolderState} themePref={themePref} setThemePref={setThemePref} onUploads={simulateUploads} />
    </div>
  );
}

/** Review-only control for switching prototype states. Not part of the product. */
function ProtoControls({ dataState, setDataState, folderState, setFolderState, themePref, setThemePref, onUploads }) {
  return (
    <Popover
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
