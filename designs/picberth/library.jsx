// 图片 and 回收站: list, filters, selection, uploads, detail.

function Thumb({ img, className = 'thumb' }) {
  return <img className={className} src={img.thumb} alt="" loading="lazy" draggable="false" />;
}

// ---------------------------------------------------------------- filters
function FilterPopover({ filters, setFilters, pool }) {
  const n = filterCount(filters);
  const tags = useMemo(() => tagCounts(pool), [pool]);
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
          <div className="fp-section">
            <div className="fp-label">标签</div>
            <div className="fp-list">
              {tags.map(([t, c]) => {
                const on = filters.tags.includes(t);
                return (
                  <button key={t} type="button" className="fp-opt" role="checkbox" aria-checked={on} onClick={() => toggle('tags', t)}>
                    <span className="check" data-on={String(on)}>{on && <Icon name="check" />}</span>
                    {t}
                    <span className="n">{c}</span>
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
  filters.tags.forEach((v) => chips.push({ k: 't' + v, label: v, rm: () => drop('tags', v) }));
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
function UploadTray({ items, onCancel, onRetry, onRemove, onCopy, onCopyAll, onClear, onOpen }) {
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
            {u.status === 'done' && <span className="s muted">{fmtBytes(u.size)}</span>}
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
function ImageTable({ list, mode, sel, openId, onRowClick, onToggle, onToggleAll, onCopy, onRestore, onTrash, onPurge, retention, selecting }) {
  const allOn = list.length > 0 && list.every((im) => sel.has(im.id));
  const someOn = list.some((im) => sel.has(im.id));
  return (
    <table className={'table' + (selecting ? ' selecting' : '')} aria-label={mode === 'trash' ? '回收站' : '图片'} aria-multiselectable="true">
      <colgroup>
        <col className="col-check" />
        <col className="col-thumb" />
        <col />
        <col className="col-tags" style={{ width: '18%' }} />
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
          <th className="col-tags">标签</th>
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
          return (
            <tr
              key={im.id}
              aria-selected={on}
              className={openId === im.id ? 'is-open' : ''}
              onClick={(e) => onRowClick(im, e)}
              tabIndex={0}
              data-id={im.id}
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
                  <span>
                    {im.width} × {im.height}
                  </span>
                  <span>{when}</span>
                </div>
              </td>
              <td className="col-tags">
                <span className="tags">
                  {im.tags.map((t) => (
                    <span key={t} className="tag">
                      {t}
                    </span>
                  ))}
                </span>
              </td>
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
                  <RowMenu im={im} mode={mode} onCopy={onCopy} onTrash={onTrash} onPurge={onPurge} />
                </div>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

function RowMenu({ im, mode, onCopy, onTrash, onPurge }) {
  return (
    <Popover
      trigger={(p) => (
        <button {...p} type="button" className="btn btn-ghost btn-sm btn-icon more" aria-label={`${im.filename} 的更多操作`} title="更多">
          <Icon name="more" />
        </button>
      )}
    >
      {(close) =>
        mode === 'active' ? (
          <>
            {LINK_FORMATS.map((f) => (
              <MenuItem key={f.key} onClick={() => (onCopy(im, f.key), close())}>
                复制 {f.label}
              </MenuItem>
            ))}
            <div className="menu-sep" />
            <MenuItem icon="download" onClick={close}>
              下载原图
            </MenuItem>
            <MenuItem icon="trash" onClick={() => (onTrash([im.id]), close())}>
              移到回收站
            </MenuItem>
          </>
        ) : (
          <MenuItem danger icon="trash" onClick={() => (onPurge([im.id]), close())}>
            永久删除
          </MenuItem>
        )
      }
    </Popover>
  );
}

function ImageGrid({ list, mode, sel, openId, onRowClick, onToggle, retention, selecting }) {
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

// ---------------------------------------------------------------- tag editor
function TagEditor({ tags, all, onChange }) {
  const [text, setText] = useState('');
  const [open, setOpen] = useState(false);
  const inRef = useRef(null);
  const fRef = useRef(null);
  const pos = useFloating(open, inRef, fRef, 'start');
  useDismiss(open, [inRef, fRef], () => setOpen(false));
  const clean = text.trim().replace(/^#/, '');
  const suggestions = all.filter(([t]) => !tags.includes(t) && t.includes(clean)).slice(0, 6);
  const add = (t) => {
    t = t.trim().replace(/^#/, '').slice(0, 32);
    if (!t || tags.includes(t) || tags.length >= 20) return;
    onChange([...tags, t]);
    setText('');
  };
  return (
    <>
      {tags.map((t) => (
        <span key={t} className="tag-edit">
          {t}
          <button type="button" aria-label={`移除标签 ${t}`} onClick={() => onChange(tags.filter((x) => x !== t))}>
            <Icon name="x" size="s" />
          </button>
        </span>
      ))}
      {tags.length < 20 && (
        <input
          ref={inRef}
          className="tag-add"
          placeholder="添加"
          aria-label="添加标签"
          role="combobox"
          aria-expanded={open && suggestions.length > 0}
          maxLength={33}
          value={text}
          onFocus={() => setOpen(true)}
          onChange={(e) => (setText(e.target.value.replace(',', '')), setOpen(true))}
          onKeyDown={(e) => {
            if (e.key === 'Enter') (e.preventDefault(), add(text));
            if (e.key === 'Backspace' && !text && tags.length) onChange(tags.slice(0, -1));
            if (e.key === 'Escape') (e.stopPropagation(), setOpen(false));
          }}
        />
      )}
      {open &&
        suggestions.length > 0 &&
        ReactDOM.createPortal(
          <div ref={fRef} className="menu" role="listbox" style={pos ? { left: pos.left, top: pos.top } : { left: -9999, top: 0 }}>
            {suggestions.map(([t, c]) => (
              <button key={t} type="button" role="option" className="menu-item" onMouseDown={(e) => e.preventDefault()} onClick={() => add(t)}>
                {t}
                <span className="end num">{c}</span>
              </button>
            ))}
          </div>,
          document.body,
        )}
    </>
  );
}

// ---------------------------------------------------------------- detail
function Detail({ img, mode, index, total, variant, onClose, onPrev, onNext, onUpdate, onTrash, onRestore, onPurge, onCopyFormat, lastFormat, allTags, retention }) {
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
          <dt>标签</dt>
          {trashed ? (
            <dd className="flex">{img.tags.length ? img.tags.map((t) => <span key={t} className="tag">{t}</span>) : <span className="muted">无</span>}</dd>
          ) : (
            <dd className="flex">
              <TagEditor tags={img.tags} all={allTags} onChange={(tags) => onUpdate({ tags })} />
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
function Library({ mode, images, setImages, uploads, uploadApi, settings, dataState, onRetryLoad, initialOpen }) {
  const toast = useToast();
  const phone = useMedia('(max-width: 720px)');
  const wide = useMedia('(min-width: 1280px)');
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

  const state = mode === 'trash' ? 'trashed' : 'active';
  const pool = useMemo(() => images.filter((im) => im.state === state), [images, state]);
  const list = useMemo(() => applyFilters(pool, filters, sort), [pool, filters, sort]);
  const allTags = useMemo(() => tagCounts(pool), [pool]);
  const openIdx = list.findIndex((im) => im.id === openId);
  const openImg = openIdx >= 0 ? list[openIdx] : null;
  const filtered = filters.q || filterCount(filters) > 0;
  const selecting = sel.size > 0 || selectMode;
  const selectedList = list.filter((im) => sel.has(im.id));

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

  const title = mode === 'trash' ? '回收站' : '图片';
  const ids = [...sel];

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
          {!phone && sel.size > 0 && <BulkActions mode={mode} ids={ids} selected={selectedList} onCopy={copyMany} onTrash={trash} onRestore={restore} onPurge={setPurge} />}
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
        <div className="head-title">
          <h1>{title}</h1>
          {dataState === 'ready' && <span className="count">{filtered ? `${list.length} / ${pool.length}` : pool.length}</span>}
        </div>
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
        <FilterPopover filters={filters} setFilters={setFilters} pool={pool} />
        <DisplayMenu view={view} setView={setView} sort={sort} setSort={setSort} phone={phone} onSelectMode={() => setSelectMode(true)} />
        {mode === 'active' && (
          <button type="button" className={'btn btn-primary' + (phone ? ' btn-icon' : '')} style={phone ? { color: 'var(--accent-ink)' } : null} aria-label="上传" title="上传 (U)" onClick={() => fileRef.current?.click()}>
            <Icon name="upload" />
            {!phone && '上传'}
          </button>
        )}
        <input ref={fileRef} type="file" accept="image/*" multiple hidden onChange={(e) => (uploadApi.add([...e.target.files]), (e.target.value = ''))} />
      </div>
    );
  }

  let content;
  if (dataState === 'loading') content = <LoadingRows />;
  else if (dataState === 'error')
    content = (
      <div className="state" role="alert">
        <span className="t">图片加载失败</span>
        <span className="d">网络连接中断</span>
        <button type="button" className="btn btn-secondary" onClick={onRetryLoad}>
          重试
        </button>
      </div>
    );
  else if (!pool.length)
    content =
      mode === 'trash' ? (
        <div className="state">
          <span className="t">回收站是空的</span>
        </div>
      ) : (
        <div className="state">
          <span className="t">还没有图片</span>
          <span className="d">拖入、粘贴或选择文件上传</span>
          <button type="button" className="btn btn-secondary" onClick={() => fileRef.current?.click()}>
            选择文件
          </button>
        </div>
      );
  else if (!list.length)
    content = (
      <div className="state">
        <span className="t">没有匹配的图片</span>
        <button type="button" className="btn btn-secondary" onClick={() => setFilters(EMPTY_FILTERS)}>
          清除筛选
        </button>
      </div>
    );
  else {
    const p = { list, mode, sel, openId, onRowClick, onToggle: toggle, retention: settings.trashRetentionDays, selecting };
    content =
      view === 'grid' ? (
        <ImageGrid {...p} />
      ) : (
        <ImageTable {...p} onToggleAll={toggleAll} onCopy={copyImg} onRestore={restore} onTrash={trash} onPurge={setPurge} />
      );
  }

  const variant = wide && !phone ? 'docked' : phone ? 'sheet' : 'drawer';

  return (
    <>
      <div className="panel-main">
        {head}
        <ActiveFilters filters={phone && searching ? { ...filters, q: '' } : filters} setFilters={setFilters} />
        <div className="scroll">
          {mode === 'active' && (
            <UploadTray
              items={uploads}
              onCancel={uploadApi.cancel}
              onRetry={uploadApi.retry}
              onRemove={uploadApi.remove}
              onClear={uploadApi.clear}
              onOpen={(id) => setOpenId(id)}
              onCopy={(id) => copyImg(images.find((im) => im.id === id))}
              onCopyAll={() => copyMany(uploads.filter((u) => u.imageId).map((u) => images.find((im) => im.id === u.imageId)).filter(Boolean))}
            />
          )}
          {content}
          {dataState === 'ready' && list.length > 12 && <div className="loadmore">已显示全部</div>}
        </div>
      </div>
      {openImg && (
        <Detail
          img={openImg}
          mode={mode}
          index={openIdx}
          total={list.length}
          variant={variant}
          retention={settings.trashRetentionDays}
          onClose={() => setOpenId(null)}
          onPrev={() => list[openIdx - 1] && setOpenId(list[openIdx - 1].id)}
          onNext={() => list[openIdx + 1] && setOpenId(list[openIdx + 1].id)}
          onUpdate={update}
          onTrash={trash}
          onRestore={restore}
          onPurge={setPurge}
          onCopyFormat={copyImg}
          lastFormat={lastFormat}
          allTags={allTags}
        />
      )}
      {phone && sel.size > 0 && (
        <div className="float-bar" role="toolbar" aria-label="批量操作">
          <BulkActions mode={mode} ids={ids} selected={selectedList} onCopy={copyMany} onTrash={trash} onRestore={restore} onPurge={setPurge} compact />
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

function BulkActions({ mode, ids, selected, onCopy, onTrash, onRestore, onPurge, compact }) {
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
      {compact && <span className="grow" />}
      <button type="button" className={'btn btn-ghost' + (compact ? ' btn-icon' : '')} aria-label="移到回收站" title="移到回收站" onClick={() => onTrash(ids)}>
        <Icon name="trash" />
        {!compact && '移到回收站'}
      </button>
    </>
  );
}

Object.assign(window, { Library });
