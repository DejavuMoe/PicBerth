// 图片 and 回收站: list, filters, selection, uploads, detail.

function Thumb({ img, className = 'thumb' }) {
  return <img className={className} src={img.thumb} alt="" loading="lazy" draggable="false" />;
}

// ---------------------------------------------------------------- filters
function FilterPopover({ filters, setFilters, pool }) {
  const n = filterCount(filters);
  const fmtCounts = useMemo(() => {
    const m = {};
    for (const im of pool) m[im.format] = (m[im.format] ?? 0) + 1;
    return m;
  }, [pool]);
  const toggle = (key, v) =>
    setFilters((f) => ({ ...f, [key]: f[key].includes(v) ? f[key].filter((x) => x !== v) : [...f[key], v] }));
  return (
    <Popover
      className="popover"
      label="筛选"
      trigger={(p) => (
        <button {...p} type="button" className="btn btn-ghost" aria-label={n ? `筛选，已选 ${n} 项` : '筛选'}>
          <Icon name="filter" />
          <span className="hide-phone">筛选</span>
          {n > 0 && <span className="num muted">{n}</span>}
        </button>
      )}
    >
      {() => (
        <>
          <div className="fp-section">
            <div className="fp-label">格式</div>
            <div className="fp-chips">
              {Object.keys(FORMAT_LABEL).map((k) => (
                <button key={k} type="button" className="chip" aria-pressed={filters.formats.includes(k)} disabled={!fmtCounts[k]} onClick={() => toggle('formats', k)}>
                  {FORMAT_LABEL[k]}
                </button>
              ))}
            </div>
          </div>
          <div className="fp-section">
            <div className="fp-label">上传时间</div>
            <div className="fp-chips">
              {DATE_RANGES.map((d) => (
                <button key={d.key} type="button" className="chip" aria-pressed={filters.date === d.key} onClick={() => setFilters((f) => ({ ...f, date: f.date === d.key ? '' : d.key }))}>
                  {d.label}
                </button>
              ))}
            </div>
          </div>
          <div className="fp-section">
            <div className="fp-label">来源</div>
            <div className="fp-list">
              {SOURCES.map((s) => {
                const on = filters.sources.includes(s.key);
                return (
                  <button key={s.key} type="button" className="fp-opt" role="checkbox" aria-checked={on} onClick={() => toggle('sources', s.key)}>
                    <span className="check" data-on={String(on)}>{on && <Icon name="check" />}</span>
                    {s.label}
                    <span className="n">{pool.filter((im) => im.source === s.key).length}</span>
                  </button>
                );
              })}
            </div>
          </div>
        </>
      )}
    </Popover>
  );
}

function ActiveFilters({ filters, setFilters }) {
  const chips = [];
  const drop = (key, v) => setFilters((f) => ({ ...f, [key]: Array.isArray(f[key]) ? f[key].filter((x) => x !== v) : '' }));
  if (filters.q) chips.push({ k: 'q', label: `“${filters.q}”`, rm: () => setFilters((f) => ({ ...f, q: '' })) });
  filters.formats.forEach((v) => chips.push({ k: 'f' + v, label: FORMAT_LABEL[v], rm: () => drop('formats', v) }));
  if (filters.date) chips.push({ k: 'date', label: DATE_RANGES.find((d) => d.key === filters.date).label, rm: () => drop('date') });
  filters.sources.forEach((v) => chips.push({ k: 's' + v, label: sourceLabel(v), rm: () => drop('sources', v) }));
  if (!chips.length) return null;
  return (
    <div className="filters">
      {chips.map((c) => (
        <span key={c.k} className="chip" style={{ color: 'var(--text)' }}>
          {c.label}
          <button type="button" className="x" aria-label={`移除 ${c.label}`} onClick={c.rm}>
            <Icon name="x" size="s" />
          </button>
        </span>
      ))}
      <button type="button" className="btn btn-ghost btn-sm muted" onClick={() => setFilters(EMPTY_FILTERS)}>
        清除
      </button>
    </div>
  );
}

function DisplayMenu({ view, setView, sort, setSort, phone, onSelectMode }) {
  return (
    <Popover
      label="显示"
      trigger={(p) => (
        <button {...p} type="button" className="btn btn-ghost btn-icon" aria-label={phone ? '更多' : '显示'} title={phone ? '更多' : '显示'}>
          <Icon name={phone ? 'more' : view === 'grid' ? 'grid' : 'list'} />
        </button>
      )}
    >
      {(close) => (
        <>
          {phone && (
            <>
              <MenuItem icon="select" onClick={() => (onSelectMode(), close())}>
                选择
              </MenuItem>
              <div className="menu-sep" />
            </>
          )}
          <div className="menu-group">视图</div>
          <MenuItem checked={view === 'table'} onClick={() => (setView('table'), close())}>
            列表
          </MenuItem>
          <MenuItem checked={view === 'grid'} onClick={() => (setView('grid'), close())}>
            网格
          </MenuItem>
          <div className="menu-sep" />
          <div className="menu-group">排序</div>
          {SORTS.map((s) => (
            <MenuItem key={s.key} checked={sort === s.key} onClick={() => (setSort(s.key), close())}>
              {s.label}
            </MenuItem>
          ))}
        </>
      )}
    </Popover>
  );
}

// ---------------------------------------------------------------- uploads
function UploadTray({ items, folders, onCancel, onRetry, onRemove, onCopy, onCopyAll, onClear, onOpen }) {
  if (!items.length) return null;
  const done = items.filter((u) => u.status === 'done' || u.status === 'duplicate');
  const busy = items.some((u) => u.status === 'queued' || u.status === 'uploading' || u.status === 'processing');
  return (
    <section className="uploads" aria-label="上传">
      <div className="uploads-head">
        <span className="strong">上传</span>
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
        {!busy && (
          <button type="button" className="btn btn-ghost btn-sm btn-icon" aria-label="关闭" title="关闭" onClick={onClear}>
            <Icon name="x" />
          </button>
        )}
      </div>
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
    </section>
  );
}

// ---------------------------------------------------------------- list views
function ImageTable({ list, mode, folders, showFolder, sel, openId, onRowClick, onToggle, onToggleAll, onCopy, onRestore, onTrash, onPurge, onMove, onDragStart, retention, selecting }) {
  const allOn = list.length > 0 && list.every((im) => sel.has(im.id));
  const someOn = list.some((im) => sel.has(im.id));
  return (
    <table className={'table' + (selecting ? ' selecting' : '')} aria-label={mode === 'trash' ? '回收站' : '图片'} aria-multiselectable="true">
      <colgroup>
        <col className="col-check" />
        <col className="col-thumb" />
        <col />
        {showFolder && <col className="col-folder" style={{ width: '20%' }} />}
        <col className="col-dims" style={{ width: 112 }} />
        <col className="col-size" style={{ width: 84 }} />
        <col className="col-date" style={{ width: 112 }} />
        <col className="col-act" style={{ width: 84 }} />
      </colgroup>
      <thead>
        <tr>
          <th className="col-check">
            <Check on={allOn ? true : someOn ? 'mixed' : false} onChange={onToggleAll} label="全选" className="cell-check" />
          </th>
          <th className="col-thumb">
            <span className="sr-only">预览</span>
          </th>
          <th>文件名</th>
          {showFolder && <th className="col-folder">文件夹</th>}
          <th className="col-dims">尺寸</th>
          <th className="col-size r">大小</th>
          <th className="col-date">{mode === 'trash' ? '清除' : '上传时间'}</th>
          <th className="col-act">
            <span className="sr-only">操作</span>
          </th>
        </tr>
      </thead>
      <tbody>
        {list.map((im) => {
          const on = sel.has(im.id);
          const when = mode === 'trash' ? purgeIn(im, retention) : fmtRel(im.createdAt);
          const where = folderLabel(folders, im.folderId);
          return (
            <tr
              key={im.id}
              aria-selected={on}
              className={openId === im.id ? 'is-open' : ''}
              onClick={(e) => onRowClick(im, e)}
              tabIndex={0}
              data-id={im.id}
              draggable={mode === 'active'}
              onDragStart={(e) => onDragStart(im, e)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') onRowClick(im, e);
                if (e.key === ' ') (e.preventDefault(), onToggle(im.id));
              }}
            >
              <td className="col-check">
                <Check on={on} onChange={(e) => onToggle(im.id, e)} label={`选择 ${im.filename}`} className="cell-check" />
              </td>
              <td className="col-thumb">
                <Thumb img={im} />
              </td>
              <td>
                <div className="name-cell">
                  <span className="name ellipsis">{im.filename}</span>
                </div>
                <div className="mobile-meta num">
                  <span>{fmtBytes(im.size)}</span>
                  {showFolder ? <span className="ellipsis">{where}</span> : (
                    <span>
                      {im.width} × {im.height}
                    </span>
                  )}
                  <span>{when}</span>
                </div>
              </td>
              {showFolder && <td className="col-folder muted">{where}</td>}
              <td className="col-dims muted num">
                {im.width} × {im.height}
              </td>
              <td className="col-size r muted num">{fmtBytes(im.size)}</td>
              <td className="col-date muted num">{when}</td>
              <td className="col-act">
                <div className="row-actions" onClick={(e) => e.stopPropagation()}>
                  {mode === 'active' ? (
                    <button type="button" className="btn btn-ghost btn-sm btn-icon" aria-label={`复制 ${im.filename} 的链接`} title="复制链接" onClick={() => onCopy(im)}>
                      <Icon name="copy" />
                    </button>
                  ) : (
                    <button type="button" className="btn btn-ghost btn-sm btn-icon" aria-label={`恢复 ${im.filename}`} title="恢复" onClick={() => onRestore([im.id])}>
                      <Icon name="restore" />
                    </button>
                  )}
                  <RowMenu im={im} mode={mode} folders={folders} onCopy={onCopy} onMove={onMove} onTrash={onTrash} onPurge={onPurge} />
                </div>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

function RowMenu({ im, mode, folders, onCopy, onMove, onTrash, onPurge }) {
  return (
    <Popover
      trigger={(p) => (
        <button {...p} type="button" className="btn btn-ghost btn-sm btn-icon more" aria-label={`${im.filename} 的更多操作`} title="更多">
          <Icon name="more" />
        </button>
      )}
    >
      {(close) => <RowMenuBody im={im} mode={mode} folders={folders} close={close} onCopy={onCopy} onMove={onMove} onTrash={onTrash} onPurge={onPurge} />}
    </Popover>
  );
}

function RowMenuBody({ im, mode, folders, close, onCopy, onMove, onTrash, onPurge }) {
  const [view, setView] = useState('main');
  if (mode !== 'active')
    return (
      <MenuItem danger icon="trash" onClick={() => (onPurge([im.id]), close())}>
        永久删除
      </MenuItem>
    );
  if (view === 'move')
    return (
      <>
        <MenuItem icon="left" onClick={() => setView('main')}>
          移动到
        </MenuItem>
        <div className="menu-sep" />
        <FolderMenuItems folders={folders} currentId={im.folderId ?? null} onPick={(id) => onMove([im.id], id)} close={close} />
      </>
    );
  return (
    <>
      {LINK_FORMATS.map((f) => (
        <MenuItem key={f.key} onClick={() => (onCopy(im, f.key), close())}>
          复制 {f.label}
        </MenuItem>
      ))}
      <div className="menu-sep" />
      <MenuItem icon="folder" onClick={() => setView('move')}>
        移动到…
      </MenuItem>
      <MenuItem icon="download" onClick={close}>
        下载原图
      </MenuItem>
      <MenuItem icon="trash" onClick={() => (onTrash([im.id]), close())}>
        移到回收站
      </MenuItem>
    </>
  );
}

function ImageGrid({ list, mode, folders, showFolder, sel, openId, onRowClick, onToggle, onDragStart, retention, selecting }) {
  return (
    <div className={'grid' + (selecting ? ' selecting' : '')} role="listbox" aria-multiselectable="true" aria-label={mode === 'trash' ? '回收站' : '图片'}>
      {list.map((im) => {
        const on = sel.has(im.id);
        return (
          <div
            key={im.id}
            className={'tile' + (openId === im.id ? ' is-open' : '')}
            role="option"
            aria-selected={on}
            tabIndex={0}
            data-id={im.id}
            draggable={mode === 'active'}
            onDragStart={(e) => onDragStart(im, e)}
            onClick={(e) => onRowClick(im, e)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') onRowClick(im, e);
              if (e.key === ' ') (e.preventDefault(), onToggle(im.id));
            }}
          >
            <Thumb img={im} className="tile-img" />
            <Check on={on} onChange={(e) => onToggle(im.id, e)} label={`选择 ${im.filename}`} />
            <div className="tile-meta">
              <span className="name ellipsis grow">{im.filename}</span>
            </div>
            <div className="tile-sub num">
              <span>{fmtBytes(im.size)}</span>
              <span>{mode === 'trash' ? purgeIn(im, retention) : fmtRel(im.createdAt)}</span>
              {showFolder && <span className="ellipsis">{folderLabel(folders, im.folderId)}</span>}
            </div>
          </div>
        );
      })}
    </div>
  );
}
function LoadingRows() {
  return (
    <div aria-busy="true" aria-label="加载中">
      {Array.from({ length: 8 }, (_, i) => (
        <div key={i} className="row" style={{ height: 48, padding: '0 20px', gap: 16, borderBottom: 'var(--bw) solid var(--hairline)' }}>
          <div className="skeleton" style={{ width: 32, height: 32, marginLeft: 36 }} />
          <div className="skeleton" style={{ width: `${30 + ((i * 17) % 30)}%`, height: 10 }} />
          <span className="grow" />
          <div className="skeleton" style={{ width: 56, height: 10 }} />
          <div className="skeleton" style={{ width: 72, height: 10, marginRight: 96 }} />
        </div>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------- folders
const DRAG_TYPE = 'application/x-picberth-images';

/** Drag-over / drop handlers for one drop target (a folder id, null = root). */
function dropHandlers(id, over, setOver, onDropImages) {
  const key = id ?? 'root';
  return {
    onDragOver: (e) => {
      if (!e.dataTransfer.types.includes(DRAG_TYPE)) return;
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      if (over !== key) setOver(key);
    },
    onDragLeave: (e) => {
      if (!e.currentTarget.contains(e.relatedTarget)) setOver(null);
    },
    onDrop: (e) => {
      const raw = e.dataTransfer.getData(DRAG_TYPE);
      setOver(null);
      if (!raw) return;
      e.preventDefault();
      onDropImages(JSON.parse(raw), id);
    },
  };
}

/** The folder tree as menu items (move targets). `blocked(id)` returns the reason a target is unavailable. */
function FolderMenuItems({ folders, currentId, onPick, close, exclude, blocked }) {
  const rows = [];
  const walk = (parent, level) =>
    childrenOf(folders, parent).forEach((f) => {
      if (exclude?.has(f.id)) return;
      rows.push([f, level]);
      walk(f.id, level + 1);
    });
  walk(null, 1);
  const pick = (id) => {
    close();
    if (id !== currentId) onPick(id);
  };
  return (
    <>
      <MenuItem checked={currentId === null} onClick={() => pick(null)}>
        {ROOT_LABEL}
      </MenuItem>
      {rows.map(([f, level]) => {
        const reason = blocked ? blocked(f.id) : '';
        return (
          <MenuItem key={f.id} indent={level + 1} checked={currentId === f.id} disabled={!!reason} end={reason || undefined} onClick={() => pick(f.id)}>
            {f.name}
          </MenuItem>
        );
      })}
    </>
  );
}

function FolderMenuBody({ folder, folders, nonEmpty, onNew, onRename, onMove, onDelete, close }) {
  const [view, setView] = useState('main');
  const full = folderDepth(folders, folder.id) >= MAX_DEPTH;
  if (view === 'move') {
    const exclude = new Set([folder.id, ...descendantIds(folders, folder.id)]);
    return (
      <>
        <MenuItem icon="left" onClick={() => setView('main')}>
          移动到
        </MenuItem>
        <div className="menu-sep" />
        <FolderMenuItems
          folders={folders}
          currentId={folder.parentId ?? null}
          exclude={exclude}
          blocked={(id) => (fitsUnder(folders, folder.id, id) ? '' : `最多 ${MAX_DEPTH} 层`)}
          onPick={onMove}
          close={close}
        />
      </>
    );
  }
  return (
    <>
      <MenuItem icon="folder-plus" disabled={full} end={full ? `最多 ${MAX_DEPTH} 层` : undefined} onClick={() => (close(), onNew())}>
        新建子文件夹
      </MenuItem>
      <MenuItem icon="edit" onClick={() => (close(), onRename())}>
        重命名
      </MenuItem>
      <MenuItem icon="folder" onClick={() => setView('move')}>
        移动到…
      </MenuItem>
      <div className="menu-sep" />
      <MenuItem danger icon="trash" disabled={nonEmpty} end={nonEmpty ? '不为空' : undefined} onClick={() => (close(), onDelete())}>
        删除
      </MenuItem>
    </>
  );
}

/** Folder tree with inline create / rename. Used as the library side pane and, below 1100 px, inside a popover. */
function FolderPanel({ folders, state, currentId, counts, onOpen, onCreate, onRename, onMove, onDelete, onDropImages, onRetry, compact, close }) {
  const [open, setOpen] = useState(() => new Set(folderChain(folders, currentId).map((f) => f.id)));
  const [creating, setCreating] = useState(undefined); // undefined: not creating, null: at root, id: under that folder
  const [renaming, setRenaming] = useState(null);
  const [draft, setDraft] = useState('');
  const [err, setErr] = useState('');
  const [over, setOver] = useState(null);

  useEffect(() => {
    setOpen((s) => new Set([...s, ...folderChain(folders, currentId).map((f) => f.id)]));
  }, [currentId]);
  useEffect(() => {
    if (!over || over === 'root' || open.has(over) || !childrenOf(folders, over).length) return;
    const t = setTimeout(() => setOpen((s) => new Set(s).add(over)), 700);
    return () => clearTimeout(t);
  }, [over]);

  const toggle = (id) => setOpen((s) => (s.has(id) ? new Set([...s].filter((x) => x !== id)) : new Set(s).add(id)));
  const cancelEdit = () => (setCreating(undefined), setRenaming(null), setErr(''));
  const startCreate = (parent) => {
    setRenaming(null);
    setDraft('');
    setErr('');
    setCreating(parent);
    if (parent) setOpen((s) => new Set(s).add(parent));
  };
  const startRename = (f) => {
    setCreating(undefined);
    setDraft(f.name);
    setErr('');
    setRenaming(f.id);
  };
  const commit = (fromBlur) => {
    const name = draft.trim();
    const cur = renaming ? folderById(folders, renaming) : null;
    if (!name && fromBlur) return cancelEdit();
    if (cur && name === cur.name) return cancelEdit();
    const e = cur ? onRename(cur.id, name) : onCreate(creating, name);
    if (!e) return cancelEdit();
    if (fromBlur) cancelEdit();
    else setErr(e);
  };

  const editor = (level) => (
    <>
      <div className="tn tn-edit" style={{ paddingLeft: 4 + (level - 1) * 16 }}>
        <span className="tn-tw" />
        <Icon name="folder" />
        <input
          className="tn-input"
          autoFocus
          aria-label="文件夹名称"
          aria-invalid={!!err}
          value={draft}
          maxLength={80}
          onFocus={(e) => e.target.select()}
          onChange={(e) => (setDraft(e.target.value), setErr(''))}
          onBlur={() => commit(true)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') (e.preventDefault(), commit(false));
            if (e.key === 'Escape') (e.stopPropagation(), cancelEdit());
          }}
        />
      </div>
      {err && (
        <div className="field-error tn-err" role="alert" style={{ paddingLeft: 4 + (level - 1) * 16 + 44 }}>
          {err}
        </div>
      )}
    </>
  );

  const row = (f, level) => {
    if (renaming === f.id) return <Fragment key={f.id}>{editor(level)}</Fragment>;
    const kids = childrenOf(folders, f.id);
    const isOpen = open.has(f.id);
    const n = counts.get(f.id) || 0;
    return (
      <Fragment key={f.id}>
        <div className={'tn' + (currentId === f.id ? ' is-cur' : '') + (over === f.id ? ' is-over' : '')} style={{ paddingLeft: 4 + (level - 1) * 16 }} {...dropHandlers(f.id, over, setOver, onDropImages)}>
          <button type="button" className="tn-tw" aria-label={isOpen ? `收起 ${f.name}` : `展开 ${f.name}`} aria-expanded={kids.length ? isOpen : undefined} disabled={!kids.length} tabIndex={kids.length ? 0 : -1} onClick={() => toggle(f.id)}>
            {kids.length > 0 && <Icon name={isOpen ? 'down' : 'right'} size="s" />}
          </button>
          <button type="button" className="tn-main" aria-current={currentId === f.id ? 'page' : undefined} onClick={() => (onOpen(f.id), close?.())}>
            <Icon name="folder" />
            <span className="ellipsis">{f.name}</span>
            {n > 0 && <span className="n num">{n}</span>}
          </button>
          <Popover
            nested={compact}
            trigger={(p) => (
              <button {...p} type="button" className="btn btn-ghost btn-icon btn-sm tn-more" aria-label={`${f.name} 的更多操作`} title="更多">
                <Icon name="more" />
              </button>
            )}
          >
            {(close2) => (
              <FolderMenuBody
                folder={f}
                folders={folders}
                nonEmpty={kids.length > 0 || n > 0}
                onNew={() => startCreate(f.id)}
                onRename={() => startRename(f)}
                onMove={(target) => onMove(f.id, target)}
                onDelete={() => onDelete(f.id)}
                close={close2}
              />
            )}
          </Popover>
        </div>
        {isOpen && kids.map((k) => row(k, level + 1))}
        {isOpen && creating === f.id && editor(level + 1)}
      </Fragment>
    );
  };

  const roots = childrenOf(folders, null);
  let body;
  if (state === 'loading')
    body = (
      <div aria-busy="true" aria-label="加载中" className="tree-skel">
        {[60, 40, 52, 36, 48].map((w, i) => (
          <div key={i} className="skeleton" style={{ width: `${w}%`, height: 10, marginLeft: 8 + (i % 3 === 2 ? 16 : 0) }} />
        ))}
      </div>
    );
  else if (state === 'error')
    body = (
      <div className="tree-note" role="alert">
        <span>文件夹加载失败</span>
        <button type="button" className="btn btn-secondary btn-sm" onClick={onRetry}>
          重试
        </button>
      </div>
    );
  else
    body = (
      <nav aria-label="文件夹">
        <div className={'tn' + (currentId === null ? ' is-cur' : '') + (over === 'root' ? ' is-over' : '')} style={{ paddingLeft: 4 }} {...dropHandlers(null, over, setOver, onDropImages)}>
          <span className="tn-tw" />
          <button type="button" className="tn-main" aria-current={currentId === null ? 'page' : undefined} onClick={() => (onOpen(null), close?.())}>
            <Icon name="image" />
            <span className="ellipsis">{ROOT_LABEL}</span>
            {counts.get(null) > 0 && <span className="n num">{counts.get(null)}</span>}
          </button>
        </div>
        {roots.map((f) => row(f, 1))}
        {creating === null && editor(1)}
        {!roots.length && creating === undefined && <div className="tree-note muted">还没有文件夹</div>}
      </nav>
    );

  return (
    <>
      <div className="tree-head">
        <span className="medium">文件夹</span>
        <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label="新建文件夹" title="新建文件夹" disabled={state !== 'ready'} onClick={() => startCreate(null)}>
          <Icon name="folder-plus" />
        </button>
      </div>
      <div className="tree-body">{body}</div>
    </>
  );
}

/** Header title: the current location. Ancestors accept dropped images. */
function Breadcrumb({ folders, currentId, count, onOpen, onDropImages, tool }) {
  const [over, setOver] = useState(null);
  const chain = folderChain(folders, currentId);
  const last = chain[chain.length - 1];
  const crumb = (f) => {
    const id = f ? f.id : null;
    return (
      <span className="crumb-anc" key={id ?? 'root'}>
        <button type="button" className={'crumb' + (over === (id ?? 'root') ? ' is-over' : '')} onClick={() => onOpen(id)} {...dropHandlers(id, over, setOver, onDropImages)}>
          {f ? f.name : ROOT_LABEL}
        </button>
        <span className="sep" aria-hidden="true">
          <Icon name="right" size="s" />
        </span>
      </span>
    );
  };
  return (
    <div className="head-title">
      {last ? (
        <nav className="crumb-trail" aria-label="位置">
          {crumb(null)}
          {chain.slice(0, -1).map(crumb)}
          <h1 className="ellipsis">{last.name}</h1>
        </nav>
      ) : (
        <h1>{ROOT_LABEL}</h1>
      )}
      {count != null && <span className="count">{count}</span>}
      {tool}
    </div>
  );
}
// ---------------------------------------------------------------- detail
function Detail({ img, mode, folders, index, total, variant, onClose, onPrev, onNext, onUpdate, onMove, onTrash, onRestore, onPurge, onCopyFormat, lastFormat, retention }) {
  const [renaming, setRenaming] = useState(false);
  const [name, setName] = useState(img.filename);
  const [copied, setCopied] = useState('');
  useEffect(() => {
    setRenaming(false);
    setName(img.filename);
    setCopied('');
  }, [img.id]);
  const touch = useRef({});
  const trashed = mode === 'trash';
  const saveName = () => {
    const n = name.trim();
    setRenaming(false);
    if (n && n !== img.filename) onUpdate({ filename: n }, '已重命名');
    else setName(img.filename);
  };
  const body = (
    <aside
      className={'detail ' + (variant === 'docked' ? '' : variant)}
      aria-label={img.filename}
      onTouchStart={(e) => (touch.current = { x: e.touches[0].clientX, y: e.touches[0].clientY })}
      onTouchEnd={(e) => {
        const dx = e.changedTouches[0].clientX - touch.current.x;
        const dy = e.changedTouches[0].clientY - touch.current.y;
        if (Math.abs(dx) > 60 && Math.abs(dx) > Math.abs(dy) * 1.5) (dx < 0 ? onNext : onPrev)();
      }}
    >
      <div className="detail-head">
        <button type="button" className="btn btn-ghost btn-icon" aria-label="上一张" title="上一张" disabled={index <= 0} onClick={onPrev}>
          <Icon name="left" />
        </button>
        <button type="button" className="btn btn-ghost btn-icon" aria-label="下一张" title="下一张" disabled={index >= total - 1} onClick={onNext}>
          <Icon name="right" />
        </button>
        <span className="pos">
          {index + 1} / {total}
        </span>
        <span className="grow" />
        <button type="button" className="btn btn-ghost btn-icon" aria-label="关闭" title="关闭" onClick={onClose}>
          <Icon name="x" />
        </button>
      </div>
      <div className="detail-body">
        <div className="preview">
          <img src={img.thumb} alt={img.filename} />
        </div>

        <div className="title-edit">
          {renaming ? (
            <input
              className="input title-input"
              aria-label="文件名"
              value={name}
              autoFocus
              onFocus={(e) => e.target.setSelectionRange(0, name.lastIndexOf('.') > 0 ? name.lastIndexOf('.') : name.length)}
              onChange={(e) => setName(e.target.value)}
              onBlur={saveName}
              onKeyDown={(e) => {
                if (e.key === 'Enter') saveName();
                if (e.key === 'Escape') (e.stopPropagation(), setName(img.filename), setRenaming(false));
              }}
            />
          ) : (
            <h2>
              <button type="button" className="title-btn" title="重命名" aria-label={`${img.filename}，重命名`} onClick={() => !trashed && setRenaming(true)} disabled={trashed}>
                {img.filename}
              </button>
            </h2>
          )}
          <div className="facts num">
            <span>
              {img.width} × {img.height}
            </span>
            <span>{fmtBytes(img.size)}</span>
            <span>{FORMAT_LABEL[img.format]}</span>
            {img.frames > 1 && <span>动图 {img.frames} 帧</span>}
            {trashed && <span className="text-warning">{purgeIn(img, retention)}</span>}
          </div>
        </div>

        {!trashed && (
          <div className="links" role="group" aria-label="复制链接">
            {LINK_FORMATS.map((f) => (
              <button
                key={f.key}
                type="button"
                className={'link-row' + (copied === f.key ? ' copied' : '')}
                aria-label={`复制 ${f.label}`}
                onClick={() => {
                  onCopyFormat(img, f.key);
                  setCopied(f.key);
                }}
              >
                <span className="k">{f.label}</span>
                <span className="v mono ellipsis">{img.links[f.key]}</span>
                <Icon name={copied === f.key ? 'check' : 'copy'} size="s" />
              </button>
            ))}
          </div>
        )}

        <dl className="props">
          <dt>文件夹</dt>
          {trashed ? (
            <dd className="ellipsis">{folderLabel(folders, img.folderId)}</dd>
          ) : (
            <dd>
              <Popover
                align="start"
                trigger={(p) => (
                  <button {...p} type="button" className="folder-btn" aria-label={`文件夹，${folderLabel(folders, img.folderId)}`}>
                    <span className="ellipsis">{folderLabel(folders, img.folderId)}</span>
                    <Icon name="down" size="s" />
                  </button>
                )}
              >
                {(close) => <FolderMenuItems folders={folders} currentId={img.folderId ?? null} onPick={(id) => onMove([img.id], id)} close={close} />}
              </Popover>
            </dd>
          )}
        </dl>

        <div className="props-divide" />

        <dl className="props">
          <dt>来源</dt>
          <dd className="ellipsis">{sourceLabel(img.source)}</dd>
          <dt>上传时间</dt>
          <dd className="num">{fmtDate(img.createdAt)}</dd>
          {trashed && (
            <>
              <dt>删除时间</dt>
              <dd className="num">{fmtDate(img.trashedAt)}</dd>
            </>
          )}
          <dt>ID</dt>
          <dd>
            <button type="button" className="copy-inline" aria-label="复制 ID" title="复制" onClick={() => onCopyFormat(img, 'id')}>
              <span className="mono ellipsis">{img.id}</span>
            </button>
          </dd>
          <dt>SHA-256</dt>
          <dd>
            <button type="button" className="copy-inline" aria-label="复制 SHA-256" title="复制" onClick={() => onCopyFormat(img, 'sha256')}>
              <span className="mono ellipsis">{img.sha256}</span>
            </button>
          </dd>
        </dl>
      </div>
      <div className="detail-foot">
        {trashed ? (
          <>
            <button type="button" className="btn btn-secondary" onClick={() => onRestore([img.id])}>
              <Icon name="restore" />
              恢复
            </button>
            <button type="button" className="btn btn-ghost btn-danger-text" onClick={() => onPurge([img.id])}>
              永久删除
            </button>
          </>
        ) : (
          <>
            <button type="button" className="btn btn-ghost">
              <Icon name="download" />
              下载
            </button>
            <button type="button" className="btn btn-ghost btn-danger-text" onClick={() => onTrash([img.id])}>
              <Icon name="trash" />
              移到回收站
            </button>
          </>
        )}
      </div>
    </aside>
  );
  if (variant === 'docked') return body;
  return ReactDOM.createPortal(
    <>
      <div className="detail-overlay" onClick={onClose} />
      {body}
    </>,
    document.body,
  );
}

// ---------------------------------------------------------------- page
function Library({ mode, images, setImages, folders, setFolders, folderState, folderPath, onNavigate, onRetryFolders, uploads, uploadApi, settings, dataState, onRetryLoad, initialOpen }) {
  const toast = useToast();
  const phone = useMedia('(max-width: 720px)');
  const wide = useMedia('(min-width: 1440px)');
  const wideNav = useMedia('(min-width: 1100px)');
  const [filters, setFilters] = useState(EMPTY_FILTERS);
  const [sort, setSort] = useState('new');
  const [view, setView] = useState(() => {
    try {
      return localStorage.getItem('picberth.view') || 'table';
    } catch (e) {
      return 'table';
    }
  });
  const [sel, setSel] = useState(() => new Set());
  const [anchor, setAnchor] = useState(null);
  const [selectMode, setSelectMode] = useState(false);
  const [openId, setOpenId] = useState(initialOpen ?? null);
  const [searching, setSearching] = useState(false);
  const [purge, setPurge] = useState(null);
  const [lastFormat, setLastFormat] = useState('url');
  const searchRef = useRef(null);
  const fileRef = useRef(null);

  const resolved = useMemo(() => (mode === 'active' && folderState === 'ready' ? resolvePath(folders, folderPath) : null), [mode, folderState, folders, folderPath]);
  const missing = resolved === undefined;
  const folderId = resolved ?? null;
  const prevFolder = useRef(folderId);

  useEffect(() => {
    try {
      localStorage.setItem('picberth.view', view);
    } catch (e) {}
  }, [view]);
  useEffect(() => {
    setSel(new Set());
    setSelectMode(false);
    setFilters(EMPTY_FILTERS);
    setOpenId(initialOpen ?? null);
  }, [mode]);
  useEffect(() => {
    if (prevFolder.current === folderId) return;
    prevFolder.current = folderId;
    setSel(new Set());
    setSelectMode(false);
    setOpenId((id) => (images.some((im) => im.id === id && (im.folderId ?? null) === folderId) ? id : null));
  }, [folderId]);

  const state = mode === 'trash' ? 'trashed' : 'active';
  const active = useMemo(() => images.filter((im) => im.state === 'active'), [images]);
  const trashedAll = useMemo(() => images.filter((im) => im.state === 'trashed'), [images]);
  const subtree = useMemo(() => (folderId === null ? null : new Set([folderId, ...descendantIds(folders, folderId)])), [folders, folderId]);
  // Browsing shows the images directly in the current folder; searching or filtering covers the folder and everything below it.
  const scope = useMemo(() => (mode === 'trash' ? trashedAll : subtree ? active.filter((im) => subtree.has(im.folderId)) : active), [mode, trashedAll, subtree, active]);
  const direct = useMemo(() => active.filter((im) => (im.folderId ?? null) === folderId), [active, folderId]);
  const filtersActive = !!filters.q.trim() || filterCount(filters) > 0;
  const base = mode === 'trash' ? trashedAll : filtersActive ? scope : direct;
  const list = useMemo(() => applyFilters(base, filters, sort), [base, filters, sort]);
  const counts = useMemo(() => {
    const m = new Map();
    for (const im of active) m.set(im.folderId ?? null, (m.get(im.folderId ?? null) || 0) + 1);
    return m;
  }, [active]);
  const showFolder = mode === 'trash' || filtersActive;
  const openIdx = list.findIndex((im) => im.id === openId);
  const openImg = openIdx >= 0 ? list[openIdx] : null;
  const filtered = filtersActive;
  const selecting = sel.size > 0 || selectMode;
  const selectedList = list.filter((im) => sel.has(im.id));
  const commonFolder = selectedList.length && selectedList.every((im) => (im.folderId ?? null) === (selectedList[0].folderId ?? null)) ? (selectedList[0].folderId ?? null) : undefined;

  // keyboard
  useEffect(() => {
    const key = (e) => {
      const typing = /INPUT|TEXTAREA/.test(document.activeElement?.tagName) || document.querySelector('.scrim');
      if (e.key === 'Escape' && !typing) {
        if (openImg) setOpenId(null);
        else if (selecting) (setSel(new Set()), setSelectMode(false));
      }
      if (typing || e.metaKey || e.ctrlKey || e.altKey) return;
      if (e.key === '/') {
        e.preventDefault();
        phone ? setSearching(true) : searchRef.current?.focus();
      }
      if ((e.key === 'u' || e.key === 'U') && mode === 'active') fileRef.current?.click();
      if (openImg && (e.key === 'ArrowLeft' || e.key === 'ArrowRight')) {
        const n = openIdx + (e.key === 'ArrowLeft' ? -1 : 1);
        if (list[n]) setOpenId(list[n].id);
      }
    };
    addEventListener('keydown', key);
    return () => removeEventListener('keydown', key);
  });
  useEffect(() => {
    if (searching) searchRef.current?.focus();
  }, [searching]);

  const fmtName = (k) => LINK_FORMATS.find((f) => f.key === k)?.label;
  const copyImg = (im, fmt = lastFormat) => {
    if (fmt === 'id' || fmt === 'sha256') {
      copyText(im[fmt]);
      toast(fmt === 'id' ? '已复制 ID' : '已复制 SHA-256');
      return;
    }
    setLastFormat(fmt);
    copyText(im.links[fmt]);
    toast(`已复制 ${fmtName(fmt)}`);
  };
  const copyMany = (ims) => {
    copyText(ims.map((im) => im.links[lastFormat]).join('\n'));
    toast(`已复制 ${ims.length} 条 ${fmtName(lastFormat)}`);
  };

  const toggle = (id, e) => {
    setSel((s) => {
      const n = new Set(s);
      if (e?.shiftKey && anchor) {
        const a = list.findIndex((im) => im.id === anchor);
        const b = list.findIndex((im) => im.id === id);
        list.slice(Math.min(a, b), Math.max(a, b) + 1).forEach((im) => n.add(im.id));
      } else n.has(id) ? n.delete(id) : n.add(id);
      return n;
    });
    setAnchor(id);
  };
  const toggleAll = () => setSel((s) => (list.every((im) => s.has(im.id)) ? new Set() : new Set(list.map((im) => im.id))));
  const onRowClick = (im, e) => {
    if (e.metaKey || e.ctrlKey || e.shiftKey || selecting) return toggle(im.id, e);
    setOpenId(openId === im.id && !wide ? null : im.id);
  };

  const patch = (ids, fn) => setImages((xs) => xs.map((im) => (ids.includes(im.id) ? fn(im) : im)));
  const advanceFrom = (ids) => {
    if (openId && ids.includes(openId)) {
      const rest = list.filter((im) => !ids.includes(im.id));
      const next = rest[Math.min(openIdx, rest.length - 1)];
      setOpenId(next ? next.id : null);
    }
  };
  const trash = (ids) => {
    const before = images;
    advanceFrom(ids);
    patch(ids, (im) => ({ ...im, state: 'trashed', trashedAt: NOW }));
    setSel(new Set());
    setSelectMode(false);
    toast(ids.length > 1 ? `已将 ${ids.length} 张图片移到回收站` : '已移到回收站', { label: '撤销', run: () => setImages(before) });
  };
  const restore = (ids) => {
    const before = images;
    advanceFrom(ids);
    patch(ids, (im) => ({ ...im, state: 'active', trashedAt: null }));
    setSel(new Set());
    setSelectMode(false);
    toast(ids.length > 1 ? `已恢复 ${ids.length} 张图片` : '已恢复', { label: '撤销', run: () => setImages(before) });
  };
  const doPurge = () => {
    const ids = purge;
    advanceFrom(ids);
    setImages((xs) => xs.filter((im) => !ids.includes(im.id)));
    setSel(new Set());
    setSelectMode(false);
    setPurge(null);
    toast(ids.length > 1 ? `已永久删除 ${ids.length} 张图片` : '已永久删除');
  };
  const update = (p, msg) => {
    patch([openImg.id], (im) => ({ ...im, ...p, links: p.filename ? makeLinks(im.id, im.ext, p.filename) : im.links }));
    if (msg) toast(msg);
  };

  // ---- folders
  const openFolder = (id) => onNavigate(libraryHash(folders, id));
  const moveImages = (ids, to) => {
    const moving = images.filter((im) => ids.includes(im.id) && im.state === 'active' && (im.folderId ?? null) !== to).map((im) => im.id);
    if (!moving.length) return;
    const before = images;
    if (!filtersActive) advanceFrom(moving);
    patch(moving, (im) => ({ ...im, folderId: to }));
    setSel(new Set());
    setSelectMode(false);
    const where = folderLabel(folders, to);
    toast(moving.length > 1 ? `已将 ${moving.length} 张图片移到 ${where}` : `已移到 ${where}`, { label: '撤销', run: () => setImages(before) });
  };
  const inside = (id) => folderId !== null && (folderId === id || descendantIds(folders, id).has(folderId));
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
    if (inside(id)) onNavigate(libraryHash(next, folderId), { replace: true });
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
    if (here) onNavigate(libraryHash(next, folderId), { replace: true });
    toast(`已将 ${f.name} 移到 ${folderLabel(folders, to)}`, {
      label: '撤销',
      run: () => (setFolders(before), here && onNavigate(libraryHash(before, folderId), { replace: true })),
    });
  };
  const deleteFolder = (id) => {
    const f = folderById(folders, id);
    if (childrenOf(folders, id).length || (counts.get(id) || 0) > 0) return toast('文件夹不为空');
    const beforeFolders = folders;
    const beforeImages = images;
    setFolders((xs) => xs.filter((x) => x.id !== id));
    setImages((xs) => xs.map((im) => (im.folderId === id ? { ...im, folderId: null } : im)));
    if (folderId === id) onNavigate(libraryHash(folders, f.parentId ?? null), { replace: true });
    toast('已删除文件夹', { label: '撤销', run: () => (setFolders(beforeFolders), setImages(beforeImages)) });
  };
  const onDragStart = (im, e) => {
    const ids = sel.has(im.id) ? [...sel] : [im.id];
    e.dataTransfer.setData(DRAG_TYPE, JSON.stringify(ids));
    e.dataTransfer.effectAllowed = 'move';
    if (ids.length > 1) {
      const g = document.createElement('div');
      g.className = 'drag-ghost';
      g.textContent = `${ids.length} 张图片`;
      document.body.appendChild(g);
      e.dataTransfer.setDragImage(g, 12, 12);
      setTimeout(() => g.remove(), 0);
    }
  };
  const panelProps = {
    folders,
    state: folderState,
    currentId: folderId,
    counts,
    onOpen: openFolder,
    onCreate: createFolder,
    onRename: renameFolder,
    onMove: moveFolder,
    onDelete: deleteFolder,
    onDropImages: moveImages,
    onRetry: onRetryFolders,
  };
  const pane = mode === 'active' && wideNav && (
    <aside className="tree-pane">
      <FolderPanel {...panelProps} />
    </aside>
  );
  if (missing)
    return (
      <>
        {pane}
        <div className="panel-main">
          <div className="head">
            <div className="head-title">
              <h1>{ROOT_LABEL}</h1>
            </div>
          </div>
          <div className="state">
            <span className="t">文件夹不存在</span>
            <button type="button" className="btn btn-secondary" onClick={() => openFolder(null)}>
              返回图片
            </button>
          </div>
        </div>
      </>
    );

  const ids = [...sel];
  const folderTool =
    mode === 'active' && !wideNav ? (
      <span className="crumb-tool">
        <Popover
          className="popover"
          align="start"
          label="文件夹"
          trigger={(p) => (
            <button {...p} type="button" className="btn btn-ghost btn-icon" aria-label="文件夹" title="文件夹">
              <Icon name="folder" />
            </button>
          )}
        >
          {(close) => <FolderPanel {...panelProps} compact close={close} />}
        </Popover>
      </span>
    ) : null;
  const countText = filtered ? `${list.length} / ${scope.length}` : base.length;

  // header
  let head;
  if (phone && searching) {
    head = (
      <div className="head">
        <div className="field-wrap grow">
          <Icon name="search" />
          <input ref={searchRef} className="input" type="search" placeholder="搜索" aria-label="搜索" value={filters.q} onChange={(e) => setFilters((f) => ({ ...f, q: e.target.value }))} />
        </div>
        <button type="button" className="btn btn-ghost" onClick={() => (setSearching(false), setFilters((f) => ({ ...f, q: '' })))}>
          取消
        </button>
      </div>
    );
  } else if (selecting) {
    head = (
      <div className="head" style={{ paddingLeft: phone ? 8 : 12 }}>
        <div className="bulk">
          <Check on={list.length && list.every((im) => sel.has(im.id)) ? true : sel.size ? 'mixed' : false} onChange={toggleAll} label="全选" />
          <span className="n">已选 {sel.size} 张</span>
          {!phone && sel.size > 0 && <BulkActions mode={mode} ids={ids} selected={selectedList} folders={folders} currentFolder={commonFolder} onCopy={copyMany} onMove={moveImages} onTrash={trash} onRestore={restore} onPurge={setPurge} />}
          <span className="grow" />
          <button type="button" className="btn btn-ghost" onClick={() => (setSel(new Set()), setSelectMode(false))}>
            {phone ? '完成' : '取消选择'}
          </button>
        </div>
      </div>
    );
  } else {
    head = (
      <div className="head">
        {mode === 'trash' ? (
          <div className="head-title">
            <h1>回收站</h1>
            {dataState === 'ready' && <span className="count">{countText}</span>}
          </div>
        ) : (
          <Breadcrumb folders={folders} currentId={folderId} count={dataState === 'ready' && folderState === 'ready' ? countText : null} onOpen={openFolder} onDropImages={moveImages} tool={folderTool} />
        )}
        {phone ? (
          <button type="button" className="btn btn-ghost btn-icon" aria-label="搜索" title="搜索" onClick={() => setSearching(true)}>
            <Icon name="search" />
          </button>
        ) : (
          <div className="field-wrap search">
            <Icon name="search" />
            <input ref={searchRef} className="input" type="search" placeholder="搜索" aria-label="搜索" value={filters.q} onChange={(e) => setFilters((f) => ({ ...f, q: e.target.value }))} />
            {!filters.q && <kbd>/</kbd>}
          </div>
        )}
        <FilterPopover filters={filters} setFilters={setFilters} pool={scope} />
        <DisplayMenu view={view} setView={setView} sort={sort} setSort={setSort} phone={phone} onSelectMode={() => setSelectMode(true)} />
        {mode === 'active' && (
          <button type="button" className={'btn btn-primary' + (phone ? ' btn-icon' : '')} style={phone ? { color: 'var(--accent-ink)' } : null} aria-label="上传" title="上传 (U)" onClick={() => fileRef.current?.click()}>
            <Icon name="upload" />
            {!phone && '上传'}
          </button>
        )}
        <input ref={fileRef} type="file" accept="image/*" multiple hidden onChange={(e) => (uploadApi.add([...e.target.files], folderId), (e.target.value = ''))} />
      </div>
    );
  }

  const loading = dataState === 'loading' || (mode === 'active' && folderState === 'loading');
  const failed = dataState === 'error' || (mode === 'active' && folderState === 'error');
  let content;
  if (loading) content = <LoadingRows />;
  else if (failed)
    content = (
      <div className="state" role="alert">
        <span className="t">{dataState === 'error' ? '图片加载失败' : '文件夹加载失败'}</span>
        <span className="d">网络连接中断</span>
        <button type="button" className="btn btn-secondary" onClick={dataState === 'error' ? onRetryLoad : onRetryFolders}>
          重试
        </button>
      </div>
    );
  else if (mode === 'trash' && !trashedAll.length)
    content = (
      <div className="state">
        <span className="t">回收站是空的</span>
      </div>
    );
  else if (!list.length && filtered)
    content = (
      <div className="state">
        <span className="t">没有匹配的图片</span>
        <button type="button" className="btn btn-secondary" onClick={() => setFilters(EMPTY_FILTERS)}>
          清除筛选
        </button>
      </div>
    );
  else if (!list.length)
    content = (
      <div className="state">
        <span className="t">{active.length ? '这里没有图片' : '还没有图片'}</span>
        <span className="d">拖入、粘贴或选择文件上传</span>
        <button type="button" className="btn btn-secondary" onClick={() => fileRef.current?.click()}>
          选择文件
        </button>
      </div>
    );
  else {
    const p = { list, mode, folders, showFolder, sel, openId, onRowClick, onToggle: toggle, onDragStart, retention: settings.trashRetentionDays, selecting };
    content =
      view === 'grid' ? (
        <ImageGrid {...p} />
      ) : (
        <ImageTable {...p} onToggleAll={toggleAll} onCopy={copyImg} onRestore={restore} onTrash={trash} onPurge={setPurge} onMove={moveImages} />
      );
  }

  const variant = wide && !phone ? 'docked' : phone ? 'sheet' : 'drawer';

  return (
    <>
      {pane}
      <div className="panel-main">
        {head}
        <ActiveFilters filters={phone && searching ? { ...filters, q: '' } : filters} setFilters={setFilters} />
        <div className="scroll">
          {mode === 'active' && (
            <UploadTray
              items={uploads}
              folders={folders}
              onCancel={uploadApi.cancel}
              onRetry={uploadApi.retry}
              onRemove={uploadApi.remove}
              onClear={uploadApi.clear}
              onOpen={(id) => {
                const im = images.find((x) => x.id === id);
                if (im && (im.folderId ?? null) !== folderId) openFolder(im.folderId ?? null);
                setOpenId(id);
              }}
              onCopy={(id) => copyImg(images.find((im) => im.id === id))}
              onCopyAll={() => copyMany(uploads.filter((u) => u.imageId).map((u) => images.find((im) => im.id === u.imageId)).filter(Boolean))}
            />
          )}
          {content}
          {dataState === 'ready' && !loading && !failed && list.length > 12 && <div className="loadmore">已显示全部</div>}
        </div>
      </div>
      {openImg && (
        <Detail
          img={openImg}
          mode={mode}
          folders={folders}
          index={openIdx}
          total={list.length}
          variant={variant}
          retention={settings.trashRetentionDays}
          onClose={() => setOpenId(null)}
          onPrev={() => list[openIdx - 1] && setOpenId(list[openIdx - 1].id)}
          onNext={() => list[openIdx + 1] && setOpenId(list[openIdx + 1].id)}
          onUpdate={update}
          onMove={moveImages}
          onTrash={trash}
          onRestore={restore}
          onPurge={setPurge}
          onCopyFormat={copyImg}
          lastFormat={lastFormat}
        />
      )}
      {phone && sel.size > 0 && (
        <div className="float-bar" role="toolbar" aria-label="批量操作">
          <BulkActions mode={mode} ids={ids} selected={selectedList} folders={folders} currentFolder={commonFolder} onCopy={copyMany} onMove={moveImages} onTrash={trash} onRestore={restore} onPurge={setPurge} compact />
        </div>
      )}
      {purge && (
        <Dialog
          title={purge.length > 1 ? `永久删除 ${purge.length} 张图片？` : `永久删除“${images.find((im) => im.id === purge[0])?.filename}”？`}
          onClose={() => setPurge(null)}
          footer={
            <>
              <button type="button" className="btn btn-secondary" data-autofocus onClick={() => setPurge(null)}>
                取消
              </button>
              <button type="button" className="btn btn-danger" onClick={doPurge}>
                永久删除
              </button>
            </>
          }
        >
          <p>文件和链接会立即失效，无法恢复。</p>
        </Dialog>
      )}
    </>
  );
}

function BulkActions({ mode, ids, selected, folders, currentFolder, onCopy, onMove, onTrash, onRestore, onPurge, compact }) {
  if (mode === 'trash')
    return (
      <>
        <button type="button" className="btn btn-ghost" onClick={() => onRestore(ids)}>
          <Icon name="restore" />
          恢复
        </button>
        {compact && <span className="grow" />}
        <button type="button" className="btn btn-ghost btn-danger-text" onClick={() => onPurge(ids)}>
          永久删除
        </button>
      </>
    );
  return (
    <>
      <button type="button" className="btn btn-ghost" onClick={() => onCopy(selected)}>
        <Icon name="copy" />
        复制链接
      </button>
      <Popover
        align="start"
        trigger={(p) => (
          <button {...p} type="button" className={'btn btn-ghost' + (compact ? ' btn-icon' : '')} aria-label="移动到文件夹" title="移动">
            <Icon name="folder" />
            {!compact && '移动'}
          </button>
        )}
      >
        {(close) => (
          <>
            <div className="menu-group">移动到</div>
            <FolderMenuItems folders={folders} currentId={currentFolder} onPick={(id) => onMove(ids, id)} close={close} />
          </>
        )}
      </Popover>
      {compact && <span className="grow" />}
      <button type="button" className={'btn btn-ghost' + (compact ? ' btn-icon' : '')} aria-label="移到回收站" title="移到回收站" onClick={() => onTrash(ids)}>
        <Icon name="trash" />
        {!compact && '移到回收站'}
      </button>
    </>
  );
}

Object.assign(window, { Library });