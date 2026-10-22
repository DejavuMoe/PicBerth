// Image management building blocks. Every component receives `lib`, the library controller created in app.jsx.

const FORMATS = ['jpeg', 'png', 'gif', 'webp', 'avif'];
const VIS_OPTS = [
  { key: 'public', label: '公开', icon: 'globe' },
  { key: 'private', label: '私有', icon: 'lock' },
];

function toggleIn(arr, v) {
  return arr.includes(v) ? arr.filter((x) => x !== v) : [...arr, v];
}
function countWhere(list, fn) {
  let n = 0;
  for (const x of list) if (fn(x)) n++;
  return n;
}
function fmtLabel(key) {
  return LINK_FORMATS.find((f) => f.key === key)?.label ?? key;
}
function stop(e) {
  e.stopPropagation();
}
const isSelecting = (lib) => lib.sel.size > 0 || lib.selectMode;

// ---------------------------------------------------------------- filters

function FilterChips({ lib }) {
  const f = lib.filters;
  const set = (p) => lib.setFilters({ ...f, ...p });
  const chips = [];
  f.formats.forEach((k) => chips.push({ k: 'f' + k, label: FORMAT_LABEL[k], rm: () => set({ formats: f.formats.filter((x) => x !== k) }) }));
  if (f.vis) chips.push({ k: 'vis', label: f.vis === 'public' ? '公开' : '私有', rm: () => set({ vis: '' }) });
  if (f.date) chips.push({ k: 'date', label: DATE_RANGES.find((d) => d.key === f.date).label, rm: () => set({ date: '' }) });
  f.sources.forEach((s) => chips.push({ k: 's' + s, label: sourceLabel(s), rm: () => set({ sources: f.sources.filter((x) => x !== s) }) }));
  f.tags.forEach((t) => chips.push({ k: 't' + t, label: `#${t}`, rm: () => set({ tags: f.tags.filter((x) => x !== t) }) }));
  if (!chips.length) return null;
  return (
    <div className="chips">
      {chips.map((c) => (
        <span className="chip" key={c.k}>
          {c.label}
          <button type="button" aria-label={`移除筛选 ${c.label}`} onClick={c.rm}>
            <Icon name="x" size={12} />
          </button>
        </span>
      ))}
      <button type="button" className="chip-clear" onClick={() => lib.setFilters({ ...EMPTY_FILTERS, q: f.q })}>
        清除筛选
      </button>
    </div>
  );
}

/** Faceted filter list, shown in the filter popover and the phone sheet. */
function FacetPanel({ lib }) {
  const f = lib.filters;
  const base = lib.base;
  const set = (p) => lib.setFilters({ ...f, ...p });
  const opt = (key, on, label, count, onClick, multi) => (
    <button type="button" key={key} className={`facet-opt ${on ? 'on' : ''}`} aria-pressed={on} onClick={onClick}>
      {multi ? (
        <span className={`menu-check ${on ? 'on' : ''}`} aria-hidden="true">
          {on && <Icon name="check" size={12} />}
        </span>
      ) : (
        <span className={`radio ${on ? 'on' : ''}`} aria-hidden="true" />
      )}
      <span className="truncate">{label}</span>
      {count !== undefined && <span className="count">{count}</span>}
    </button>
  );
  const head = (title, active, clear) => (
    <h3>
      {title}
      {active && (
        <button type="button" onClick={clear}>
          清除
        </button>
      )}
    </h3>
  );
  return (
    <div>
      <div className="facet">
        {head('格式', f.formats.length > 0, () => set({ formats: [] }))}
        {FORMATS.map((k) => opt(k, f.formats.includes(k), FORMAT_LABEL[k], countWhere(base, (i) => i.format === k), () => set({ formats: toggleIn(f.formats, k) }), true))}
      </div>
      <div className="facet">
        {head('可见性', !!f.vis, () => set({ vis: '' }))}
        {opt('all', !f.vis, '全部', base.length, () => set({ vis: '' }))}
        {VIS_OPTS.map((o) => opt(o.key, f.vis === o.key, o.label, countWhere(base, (i) => i.visibility === o.key), () => set({ vis: f.vis === o.key ? '' : o.key })))}
      </div>
      <div className="facet">
        {head('上传时间', !!f.date, () => set({ date: '' }))}
        {opt('all', !f.date, '不限', base.length, () => set({ date: '' }))}
        {DATE_RANGES.map((d) =>
          opt(d.key, f.date === d.key, d.label, countWhere(base, (i) => (d.since ? i.createdAt >= d.since() : i.createdAt < d.until())), () => set({ date: f.date === d.key ? '' : d.key })),
        )}
      </div>
      <div className="facet">
        {head('来源', f.sources.length > 0, () => set({ sources: [] }))}
        {SOURCES.map((s) => opt(s.key, f.sources.includes(s.key), s.label, countWhere(base, (i) => i.source === s.key), () => set({ sources: toggleIn(f.sources, s.key) }), true))}
      </div>
      <div className="facet">
        {head('标签', f.tags.length > 0, () => set({ tags: [] }))}
        {allTags(base).map(([t, n]) => opt(t, f.tags.includes(t), t, n, () => set({ tags: toggleIn(f.tags, t) }), true))}
      </div>
    </div>
  );
}

/** The single 筛选 button: every facet lives in one popover. */
function FilterPopover({ lib }) {
  const n = activeFilterCount({ ...lib.filters, q: '' }, true);
  return (
    <Menu
      align="right"
      className="menu-facets"
      width={300}
      trigger={(open, toggle) => (
        <button type="button" className="btn btn-ghost filter-btn" aria-haspopup="menu" aria-expanded={open} onClick={toggle}>
          <Icon name="filter" />
          <span>筛选</span>
          {n > 0 && <span className="filter-n">{n}</span>}
        </button>
      )}
    >
      <FacetPanel lib={lib} />
    </Menu>
  );
}

/** Phone adjustments sheet: sort, view and the same facets, applied live. */
function AdjustSheet({ lib, onClose }) {
  const n = activeFilterCount({ ...lib.filters, q: '' }, true);
  return (
    <Sheet onClose={onClose} label="筛选">
      {(close) => (
        <>
          <div className="sheet-head">
            <h2>筛选</h2>
            {n > 0 && (
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => lib.setFilters({ ...EMPTY_FILTERS, q: lib.filters.q })}>
                清除
              </button>
            )}
          </div>
          <div className="sheet-body">
            <div className="facet">
              <h3>排序</h3>
              {SORTS.map((s) => (
                <button type="button" key={s.key} className={`facet-opt ${lib.sort === s.key ? 'on' : ''}`} aria-pressed={lib.sort === s.key} onClick={() => lib.setSort(s.key)}>
                  <span className={`radio ${lib.sort === s.key ? 'on' : ''}`} aria-hidden="true" />
                  {s.label}
                </button>
              ))}
            </div>
            <div className="facet">
              <h3>视图</h3>
              <div className="facet-seg">
                <Seg
                  fill
                  label="视图"
                  value={lib.view}
                  onChange={lib.setView}
                  options={[
                    { key: 'table', icon: 'list', label: '列表' },
                    { key: 'grid', icon: 'grid', label: '网格' },
                  ]}
                />
              </div>
            </div>
            <FacetPanel lib={lib} />
          </div>
          <div className="sheet-foot">
            <button type="button" className="btn btn-primary btn-block btn-lg" onClick={close}>
              显示 {lib.items.length} 张图片
            </button>
          </div>
        </>
      )}
    </Sheet>
  );
}

function SortMenu({ lib }) {
  const cur = SORTS.find((s) => s.key === lib.sort) ?? SORTS[0];
  return (
    <Menu
      align="right"
      width={160}
      trigger={(open, toggle) => (
        <button type="button" className="btn btn-ghost btn-icon" aria-haspopup="menu" aria-expanded={open} aria-label={`排序：${cur.label}`} title={`排序：${cur.label}`} onClick={toggle}>
          <Icon name="sort" />
        </button>
      )}
    >
      {(close) =>
        SORTS.map((s) => (
          <RadioItem key={s.key} on={s.key === lib.sort} onClick={() => (lib.setSort(s.key), close())}>
            {s.label}
          </RadioItem>
        ))
      }
    </Menu>
  );
}

function ViewToggle({ lib }) {
  return (
    <Seg
      icons
      label="视图"
      value={lib.view}
      onChange={lib.setView}
      options={[
        { key: 'table', icon: 'list', title: '列表' },
        { key: 'grid', icon: 'grid', title: '网格' },
      ]}
    />
  );
}

/** Search field. `collapsed`: an icon button that expands into the field (tablet and phone headers). */
function SearchBox({ lib, inputRef, collapsed }) {
  const q = lib.filters.q;
  if (collapsed && !lib.searchOpen && !q)
    return (
      <button type="button" className="btn btn-ghost btn-icon" aria-label="搜索" title="搜索 (/)" onClick={() => lib.setSearchOpen(true)}>
        <Icon name="search" />
      </button>
    );
  return (
    <label className={`search ${collapsed ? 'search-open' : ''}`}>
      <Icon name="search" />
      <input
        ref={inputRef}
        className="input"
        type="text"
        role="searchbox"
        autoComplete="off"
        spellCheck="false"
        placeholder="搜索"
        aria-label="搜索图片"
        autoFocus={collapsed}
        value={q}
        onChange={(e) => lib.setFilters({ ...lib.filters, q: e.target.value })}
        onBlur={() => {
          if (collapsed && !lib.filters.q && !lib.isMobile) lib.setSearchOpen(false);
        }}
        onKeyDown={(e) => {
          if (e.key !== 'Escape') return;
          lib.setFilters({ ...lib.filters, q: '' });
          lib.setSearchOpen(false);
          e.currentTarget.blur();
        }}
      />
      {q ? (
        <button
          type="button"
          className="search-clear"
          aria-label="清除"
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => {
            lib.setFilters({ ...lib.filters, q: '' });
            inputRef.current?.focus();
          }}
        >
          <Icon name="x" size={12} />
        </button>
      ) : (
        !collapsed && <kbd className="hide-touch">/</kbd>
      )}
    </label>
  );
}

// ---------------------------------------------------------------- page header

/** Library header. Desktop and tablet: title left, tools right (replaced in place by the bulk actions while selecting).
 *  Phone: compact title row with search, select mode and the adjustments sheet. */
function LibraryHead({ lib, width, onUpload, onAdjust }) {
  const trash = lib.route === 'trash';
  const title = trash ? '回收站' : '全部图片';
  const selecting = isSelecting(lib);
  const hasTools = !((lib.dataState === 'ok' || lib.dataState === 'empty') && !lib.base.length);

  if (lib.isMobile) {
    const n = activeFilterCount({ ...lib.filters, q: '' }, true);
    if (lib.searchOpen || lib.filters.q)
      return (
        <header className="page-head">
          <SearchBox lib={lib} inputRef={lib.searchRef} collapsed />
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => {
              lib.setFilters({ ...lib.filters, q: '' });
              lib.setSearchOpen(false);
            }}
          >
            取消
          </button>
        </header>
      );
    return (
      <header className="page-head">
        <h1>{title}</h1>
        {hasTools && (
          <div className="page-actions">
            <SearchBox lib={lib} inputRef={lib.searchRef} collapsed />
            <button type="button" className={`btn btn-ghost btn-icon ${selecting ? 'active-soft' : ''}`} aria-label="选择" aria-pressed={selecting} onClick={() => (selecting ? lib.clearSel() : lib.setSelectMode(true))}>
              <Icon name="select" />
            </button>
            <button type="button" className={`btn btn-ghost btn-icon ${n ? 'active-soft' : ''}`} aria-label={n ? `筛选（${n}）` : '筛选'} onClick={onAdjust}>
              <Icon name="more" />
            </button>
          </div>
        )}
      </header>
    );
  }

  const w = width || 1200;
  const collapsed = lib.rail || w < (trash ? 660 : 640);
  return (
    <PageHead title={title}>
      {selecting ? (
        <BulkBar lib={lib} compact={w < 620} />
      ) : (
        <>
          {hasTools && (
            <>
              <SearchBox lib={lib} inputRef={lib.searchRef} collapsed={collapsed} />
              <FilterPopover lib={lib} />
              <SortMenu lib={lib} />
              <ViewToggle lib={lib} />
            </>
          )}
          {trash ? (
            lib.base.length > 0 && (
              <button type="button" className="btn btn-danger-ghost" onClick={() => lib.act.purge(lib.base.map((i) => i.id))}>
                清空回收站
              </button>
            )
          ) : (
            hasTools && (
              <button type="button" className="btn btn-primary" onClick={onUpload} title="上传 (U)">
                <Icon name="upload" />
                上传
              </button>
            )
          )}
        </>
      )}
    </PageHead>
  );
}

// ---------------------------------------------------------------- selection

function copyMany(lib, imgs, fmt = lib.linkFormat) {
  writeClipboard(imgs.map((i) => i.links[fmt]).join('\n'));
  lib.toast(imgs.length === 1 ? `已复制 ${fmtLabel(fmt)} 链接` : `已复制 ${imgs.length} 条 ${fmtLabel(fmt)} 链接`);
}

/** Bulk actions. In the page header on desktop and tablet; `floating` above the tab bar on phones.
 *  `compact`: icon-only buttons. */
function BulkBar({ lib, compact, floating }) {
  const ids = [...lib.sel];
  const imgs = ids.map((id) => lib.byId[id]).filter(Boolean);
  const trash = lib.route === 'trash';
  const all = lib.items.length;
  const allOn = ids.length === all && all > 0;
  const sm = floating ? 'btn-sm' : '';
  const aria = (text, extra) => ({ 'aria-label': compact ? text : undefined, title: compact ? text : undefined, ...extra });
  return (
    <div className={`bulk ${floating ? 'bulk-floating' : ''}`} role="toolbar" aria-label="批量操作">
      <span className="count">{ids.length ? `已选 ${ids.length} 张` : '点按图片以选择'}</span>
      <button type="button" className={`btn btn-ghost ${sm}`} onClick={allOn ? lib.clearSel : lib.selectAll}>
        {allOn ? '取消全选' : '全选'}
      </button>
      {floating && <span className="grow" />}
      {ids.length > 0 &&
        (trash ? (
          <>
            <button type="button" className={`btn btn-ghost ${sm}`} onClick={() => lib.act.restore(ids)}>
              <Icon name="restore" />
              恢复
            </button>
            <button type="button" className={`btn btn-danger-ghost ${sm}`} onClick={() => lib.act.purge(ids)}>
              永久删除
            </button>
          </>
        ) : (
          <>
            <button type="button" className={`btn btn-ghost ${sm} ${compact ? 'btn-icon' : ''}`} {...aria('复制链接')} onClick={() => copyMany(lib, imgs)}>
              <Icon name="copy" />
              {!compact && '复制链接'}
            </button>
            <Menu
              align="right"
              up={floating}
              trigger={(open, toggle) => (
                <button type="button" className={`btn btn-ghost ${sm} ${compact ? 'btn-icon' : ''}`} {...aria('可见性', { 'aria-expanded': open, 'aria-haspopup': 'menu' })} onClick={toggle}>
                  <Icon name="lock" />
                  {!compact && '可见性'}
                  {!compact && <Icon name="chevron-down" size={14} />}
                </button>
              )}
            >
              {(close) => (
                <>
                  <MenuItem icon="globe" onClick={() => (lib.act.setVis(ids, 'public'), close())}>
                    设为公开
                  </MenuItem>
                  <MenuItem icon="lock" onClick={() => (lib.act.setVis(ids, 'private'), close())}>
                    设为私有
                  </MenuItem>
                </>
              )}
            </Menu>
            <button type="button" className={`btn btn-danger-ghost ${sm} ${compact ? 'btn-icon' : ''}`} {...aria('移到回收站')} onClick={() => lib.act.trash(ids)}>
              <Icon name="trash" />
              {!compact && '移到回收站'}
            </button>
          </>
        ))}
      <button type="button" className={`btn btn-ghost btn-icon ${sm}`} aria-label="取消选择" title="取消选择 (Esc)" onClick={lib.clearSel}>
        <Icon name="x" />
      </button>
    </div>
  );
}

// ---------------------------------------------------------------- per-image actions

function RowMenu({ lib, image, up }) {
  const trash = image.state === 'trashed';
  return (
    <span onClick={stop}>
      <Menu
        align="right"
        up={up}
        width={196}
        trigger={(open, toggle) => (
          <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label={`${image.filename} 的更多操作`} title="更多操作" aria-expanded={open} onClick={toggle}>
            <Icon name="more" />
          </button>
        )}
      >
        {(close) => {
          const run = (fn) => () => {
            close();
            fn();
          };
          return trash ? (
            <>
              <MenuItem icon="restore" onClick={run(() => lib.act.restore([image.id]))}>
                恢复
              </MenuItem>
              <MenuItem icon="download" onClick={run(() => lib.act.download(image))}>
                下载
              </MenuItem>
              <div className="menu-sep" />
              <MenuItem icon="trash" danger onClick={run(() => lib.act.purge([image.id]))}>
                永久删除
              </MenuItem>
            </>
          ) : (
            <>
              <div className="menu-label">复制为</div>
              {LINK_FORMATS.map((f) => (
                <RadioItem
                  key={f.key}
                  on={f.key === lib.linkFormat}
                  onClick={run(() => {
                    lib.setLinkFormat(f.key);
                    copyMany(lib, [image], f.key);
                  })}
                >
                  {f.label}
                </RadioItem>
              ))}
              <div className="menu-sep" />
              <MenuItem icon="external" onClick={run(() => lib.toast('已在新标签页打开原图'))}>
                打开原图
              </MenuItem>
              <MenuItem icon="download" onClick={run(() => lib.act.download(image))}>
                下载
              </MenuItem>
              {image.visibility === 'public' ? (
                <MenuItem icon="lock" onClick={run(() => lib.act.setVis([image.id], 'private'))}>
                  设为私有
                </MenuItem>
              ) : (
                <MenuItem icon="globe" onClick={run(() => lib.act.setVis([image.id], 'public'))}>
                  设为公开
                </MenuItem>
              )}
              <div className="menu-sep" />
              <MenuItem icon="trash" danger onClick={run(() => lib.act.trash([image.id]))}>
                移到回收站
              </MenuItem>
            </>
          );
        }}
      </Menu>
    </span>
  );
}

function CopyLink({ lib, image, className = 'btn-ghost' }) {
  return <CopyButton iconOnly size="sm" className={className} text={image.links[lib.linkFormat]} label={`复制 ${fmtLabel(lib.linkFormat)}`} onCopied={() => lib.toast(`已复制 ${fmtLabel(lib.linkFormat)} 链接`)} />;
}

function LockMark({ image }) {
  return image.visibility === 'private' ? (
    <span className="lock-mark" role="img" aria-label="私有" title="私有">
      <Icon name="lock" size={13} />
    </span>
  ) : null;
}

// ---------------------------------------------------------------- list (desktop and tablet)

function ImageList({ lib, items }) {
  const trash = lib.route === 'trash';
  return (
    <div className="list-wrap">
      <div className={`list ${isSelecting(lib) ? 'selecting' : ''}`} role="grid" aria-multiselectable="true" aria-label={trash ? '回收站' : '全部图片'}>
        {items.map((im, idx) => {
          const on = lib.sel.has(im.id);
          return (
            <div
              key={im.id}
              className={`lrow ${trash ? 'trash' : ''} ${on ? 'selected' : ''} ${lib.activeId === im.id ? 'active' : ''}`}
              role="row"
              aria-selected={on}
              tabIndex={0}
              onClick={(e) => lib.clickItem(im.id, e)}
              onKeyDown={(e) => {
                if (e.target !== e.currentTarget) return;
                if (e.key === 'Enter') lib.open(im.id);
                if (e.key === ' ') {
                  e.preventDefault();
                  lib.toggle(im.id);
                }
              }}
            >
              <span role="gridcell" className="l-check" onClick={stop}>
                <input type="checkbox" className="check" checked={on} onChange={() => {}} onClick={(e) => lib.toggle(im.id, e.shiftKey)} aria-label={`选择 ${im.filename}`} />
              </span>
              <span role="gridcell" className="l-thumb">
                <Thumb image={im} className="thumb-sm" />
              </span>
              <span role="gridcell" className="l-name">
                <span className="name truncate" title={im.filename}>
                  {im.filename}
                </span>
                <LockMark image={im} />
                {im.tags.length > 0 && (
                  <span className="tags">
                    {im.tags.map((t) => (
                      <span key={t} className="tag">
                        {t}
                      </span>
                    ))}
                  </span>
                )}
              </span>
              {!trash && (
                <span role="gridcell" className="l-dim">
                  {im.width}
                  {' × '}
                  {im.height}
                </span>
              )}
              <span role="gridcell" className="l-size">
                {fmtBytes(im.size)}
              </span>
              {trash ? (
                <>
                  <span role="gridcell" className="l-time" title={fmtDate(im.trashedAt)}>
                    {fmtRel(im.trashedAt)}
                  </span>
                  <span role="gridcell" className="l-time l-last">
                    {daysLeft(im, lib.settings.trashRetentionDays)}
                  </span>
                </>
              ) : (
                <span role="gridcell" className="l-time l-last" title={fmtDate(im.createdAt)}>
                  {fmtRel(im.createdAt)}
                </span>
              )}
              <span role="gridcell" className="l-actions">
                {trash ? (
                  <button
                    type="button"
                    className="btn btn-ghost btn-sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      lib.act.restore([im.id]);
                    }}
                  >
                    恢复
                  </button>
                ) : (
                  <CopyLink lib={lib} image={im} />
                )}
                <RowMenu lib={lib} image={im} up={items.length > 5 && idx >= items.length - 3} />
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
// ---------------------------------------------------------------- grid

function Tile({ lib, image }) {
  const on = lib.sel.has(image.id);
  const trash = image.state === 'trashed';
  return (
    <div
      className={`tile ${on ? 'selected' : ''} ${lib.activeId === image.id ? 'active' : ''}`}
      onClick={(e) => lib.clickItem(image.id, e)}
      tabIndex={0}
      role="button"
      aria-label={image.filename}
      aria-pressed={on}
      onKeyDown={(e) => {
        if (e.target !== e.currentTarget) return;
        if (e.key === 'Enter') lib.open(image.id);
        if (e.key === ' ') {
          e.preventDefault();
          lib.toggle(image.id);
        }
      }}
      {...longPress(lib, image.id)}
    >
      <span className="tile-media">
        <Thumb image={image} />
        <label className="tile-check" onClick={stop}>
          <input type="checkbox" className="check" checked={on} onChange={() => {}} onClick={(e) => lib.toggle(image.id, e.shiftKey)} aria-label={`选择 ${image.filename}`} />
        </label>
        <span className="tile-badges">
          {image.frames > 1 && <span className="chip-overlay">GIF</span>}
          {image.visibility === 'private' && (
            <span className="chip-overlay" role="img" aria-label="私有" title="私有">
              <Icon name="lock" size={12} />
            </span>
          )}
        </span>
        {!trash && (
          <span className="tile-copy" onClick={stop}>
            <CopyLink lib={lib} image={image} className="btn-overlay" />
          </span>
        )}
      </span>
      <span className="name truncate" title={image.filename}>
        {image.filename}
      </span>
    </div>
  );
}

function ImageGrid({ lib, items }) {
  return (
    <div className={`grid ${isSelecting(lib) ? 'selecting' : ''}`}>
      {items.map((im) => (
        <Tile key={im.id} lib={lib} image={im} />
      ))}
    </div>
  );
}

// ---------------------------------------------------------------- phone rows

/** Touch long-press enters selection, mirroring the production gesture. */
function longPress(lib, id) {
  let timer = 0;
  const cancel = () => clearTimeout(timer);
  return {
    onPointerDown: (e) => {
      if (e.pointerType !== 'touch') return;
      timer = setTimeout(() => {
        lib.setSelectMode(true);
        lib.toggle(id);
        lib.suppressClick();
        navigator.vibrate?.(10);
      }, 450);
    },
    onPointerUp: cancel,
    onPointerLeave: cancel,
    onPointerMove: cancel,
  };
}

function ImageRows({ lib, items }) {
  const selecting = isSelecting(lib);
  const trash = lib.route === 'trash';
  return (
    <div className={`rows ${selecting ? 'selecting' : ''}`}>
      {items.map((im) => {
        const on = lib.sel.has(im.id);
        return (
          <div key={im.id} className={`row ${on ? 'selected' : ''} ${lib.activeId === im.id ? 'active' : ''}`} {...longPress(lib, im.id)}>
            <span className="row-check" aria-hidden={!selecting}>
              <input type="checkbox" className="check" checked={on} tabIndex={selecting ? 0 : -1} onChange={() => lib.toggle(im.id)} aria-label={`选择 ${im.filename}`} />
            </span>
            <button type="button" className="row-btn" onClick={(e) => lib.clickItem(im.id, e)}>
              <Thumb image={im} />
              <span className="l-name">
                <span className="name truncate">{im.filename}</span>
                <LockMark image={im} />
              </span>
              <span className="row-time num">{trash ? daysLeft(im, lib.settings.trashRetentionDays) : fmtRel(im.createdAt)}</span>
            </button>
            {!selecting && (trash ? <RowMenu lib={lib} image={im} /> : <CopyLink lib={lib} image={im} />)}
          </div>
        );
      })}
    </div>
  );
}

// ---------------------------------------------------------------- collection with states

function LoadingList({ grid, mobile }) {
  if (grid)
    return (
      <div className="grid" aria-busy="true" aria-label="正在加载">
        {Array.from({ length: 12 }, (_, i) => (
          <div key={i} className="tile is-skeleton">
            <div className="skeleton" style={{ aspectRatio: '4 / 3', borderRadius: 0 }} />
            <div className="skeleton" style={{ height: 12, width: '70%', margin: 8 }} />
          </div>
        ))}
      </div>
    );
  return (
    <div className={`rows rows-skeleton ${mobile ? '' : 'desk'}`} aria-busy="true" aria-label="正在加载">
      {Array.from({ length: 8 }, (_, i) => (
        <div key={i} className="row">
          <div className="skeleton" style={{ width: mobile ? 36 : 28, height: mobile ? 36 : 28, flex: 'none', borderRadius: 'var(--r-xs)' }} />
          <div className="skeleton" style={{ height: 12, width: `${30 + ((i * 17) % 35)}%` }} />
        </div>
      ))}
    </div>
  );
}

function EmptyState({ lib }) {
  if (lib.route === 'trash')
    return (
      <div className="empty">
        <Icon name="trash" size={32} />
        <strong>回收站是空的</strong>
      </div>
    );
  return (
    <div className="empty">
      <Icon name="upload" size={32} />
      <strong>还没有图片</strong>
      <div className="actions">
        <button type="button" className="btn btn-primary" onClick={lib.pick}>
          <Icon name="upload" />
          上传图片
        </button>
        <button type="button" className="btn btn-secondary" onClick={() => lib.go('tokens')}>
          <Icon name="key" />
          新建令牌
        </button>
      </div>
    </div>
  );
}

function ImageCollection({ lib }) {
  const grid = lib.view === 'grid';
  if (lib.dataState === 'loading') return <LoadingList grid={grid} mobile={lib.isMobile} />;
  if (lib.dataState === 'error')
    return (
      <div className="alert danger" role="alert" style={{ margin: 16 }}>
        <Icon name="alert" />
        <div className="grow">
          <strong>无法加载图片列表</strong>
          <div>无法连接到服务器</div>
        </div>
        <button type="button" className="btn btn-secondary btn-sm" onClick={lib.retry}>
          <Icon name="refresh" />
          重试
        </button>
      </div>
    );
  if (!lib.base.length) return <EmptyState lib={lib} />;
  if (!lib.items.length)
    return (
      <div className="empty">
        <Icon name="search" size={32} />
        <strong>没有符合条件的图片</strong>
        <div className="actions">
          <button type="button" className="btn btn-secondary" onClick={() => lib.setFilters(EMPTY_FILTERS)}>
            清除搜索和筛选
          </button>
        </div>
      </div>
    );
  return grid ? <ImageGrid lib={lib} items={lib.items} /> : lib.isMobile ? <ImageRows lib={lib} items={lib.items} /> : <ImageList lib={lib} items={lib.items} />;
}

// ---------------------------------------------------------------- detail

/** Filename heading; click it (or the pencil) to rename in place. */
function DetailName({ image, editable, onSave }) {
  const [edit, setEdit] = React.useState(false);
  const [v, setV] = React.useState(image.filename);
  const inputRef = React.useRef(null);
  React.useEffect(() => {
    setEdit(false);
    setV(image.filename);
  }, [image.id, image.filename]);
  // Entering edit mode focuses the field and selects the name without its extension.
  React.useEffect(() => {
    const el = inputRef.current;
    if (!edit || !el) return;
    el.focus();
    const dot = el.value.lastIndexOf('.');
    el.setSelectionRange(0, dot > 0 ? dot : el.value.length);
  }, [edit]);
  const err = !v.trim() ? '请输入文件名' : v.length > 200 ? '文件名不能超过 200 个字符' : null;
  if (!edit)
    return (
      <h2 className="d-name">
        {editable ? (
          <button type="button" className="d-name-btn" title="重命名" aria-label={`重命名 ${image.filename}`} onClick={() => setEdit(true)}>
            <span>{image.filename}</span>
            <Icon name="edit" size={14} />
          </button>
        ) : (
          <span>{image.filename}</span>
        )}
      </h2>
    );
  const cancel = () => {
    setEdit(false);
    setV(image.filename);
  };
  const save = () => {
    if (err) return;
    onSave(v.trim());
    setEdit(false);
  };
  return (
    <div className="d-rename">
      <div className="rename">
        <input
          ref={inputRef}
          className="input"
          value={v}
          aria-label="文件名"
          aria-invalid={!!err}
          onChange={(e) => setV(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') save();
            if (e.key === 'Escape') {
              e.stopPropagation();
              e.nativeEvent.stopImmediatePropagation();
              cancel();
            }
          }}
        />
        <button type="button" className="btn btn-primary" disabled={!!err} onClick={save}>
          保存
        </button>
        <button type="button" className="btn btn-ghost" onClick={cancel}>
          取消
        </button>
      </div>
      {err && <p className="error-text">{err}</p>}
    </div>
  );
}

const normalizeTag = (s) => s.trim().replace(/^#+/, '').trim().slice(0, 32);

/** Tag chips plus a `+` chip that opens the product combobox (existing tags with counts, or the typed text as a new tag). */
function TagEditor({ image, onChange, known }) {
  const tags = image.tags;
  const full = tags.length >= 20;
  const options = known.filter(([t]) => !tags.includes(t)).map(([t, n]) => ({ value: t, label: t, count: n }));
  return (
    <div className="tag-editor">
      {tags.map((t) => (
        <span key={t} className="tag tag-edit">
          {t}
          <button type="button" aria-label={`移除标签 ${t}`} onClick={() => onChange(tags.filter((x) => x !== t))}>
            <Icon name="x" size={12} />
          </button>
        </span>
      ))}
      <Combobox
        label="添加标签"
        placeholder="添加标签"
        options={options}
        normalize={normalizeTag}
        taken={(v) => tags.includes(v)}
        onPick={(v) => !tags.includes(v) && tags.length < 20 && onChange([...tags, v])}
        onEmptyBackspace={() => tags.length && onChange(tags.slice(0, -1))}
        trigger={(open, toggle) => (
          <button type="button" className="tag-add" aria-label="添加标签" aria-haspopup="listbox" aria-expanded={open} disabled={full} onClick={toggle}>
            <Icon name="plus" size={12} />
          </button>
        )}
      />
    </div>
  );
}

function KeyValue({ k, children, top }) {
  return (
    <>
      <dt className={top ? 'top' : ''}>{k}</dt>
      <dd className={top ? 'top' : ''}>{children}</dd>
    </>
  );
}

/** `mode`: dock (column beside the list), drawer (overlay on tablets) or sheet (phones). */
function DetailPanel({ lib, image, onClose, mode = 'drawer' }) {
  const trash = image.state === 'trashed';
  const [prev, next] = lib.neighbors(image.id);
  const pub = image.visibility === 'public';
  const fmt = lib.linkFormat;
  return (
    <div className={`detail detail-${mode}`}>
      <div className="d-top">
        <button type="button" className="btn btn-ghost btn-icon" aria-label="上一张" title="上一张 (←)" disabled={!prev} onClick={() => prev && lib.open(prev.id)}>
          <Icon name="chevron-left" />
        </button>
        <button type="button" className="btn btn-ghost btn-icon" aria-label="下一张" title="下一张 (→)" disabled={!next} onClick={() => next && lib.open(next.id)}>
          <Icon name="chevron-right" />
        </button>
        <span className="sp" />
        <button type="button" className="btn btn-ghost btn-icon" aria-label="关闭" title="关闭 (Esc)" onClick={onClose}>
          <Icon name="x" />
        </button>
      </div>
      <div className="detail-scroll">
        <div className={`preview ${hasAlpha(image) ? 'checker' : ''}`}>
          <a
            className="preview-img"
            href={image.url}
            target="_blank"
            rel="noreferrer"
            aria-label="打开原图"
            tabIndex={-1}
            onClick={(e) => {
              e.preventDefault();
              lib.toast('已在新标签页打开原图');
            }}
          >
            <img key={image.id} src={image.thumb} alt={image.filename} />
          </a>
          <a
            className="preview-open"
            href={image.url}
            target="_blank"
            rel="noreferrer"
            onClick={(e) => {
              e.preventDefault();
              lib.toast('已在新标签页打开原图');
            }}
          >
            <Icon name="external" size={14} />
            打开原图
          </a>
        </div>

        <div className="d-body">
          <div>
            <DetailName image={image} editable={!trash} onSave={(n) => lib.act.rename(image.id, n)} />
            <div className="d-meta num">
              <span>
                {FORMAT_LABEL[image.format]} · {image.width} × {image.height} · {fmtBytes(image.size)} ·{' '}
                <time title={fmtDate(image.createdAt)}>{fmtRel(image.createdAt)}</time>
              </span>
              {trash && <span className="badge">{daysLeft(image, lib.settings.trashRetentionDays)}</span>}
            </div>
          </div>

          {!trash && (
            <div className="d-link">
              <Tabs label="链接格式" value={fmt} onChange={lib.setLinkFormat} options={LINK_FORMATS} />
              <div className="link-box">
                <input className="input mono" readOnly value={image.links[fmt]} aria-label={`${fmtLabel(fmt)} 链接`} onFocus={(e) => e.target.select()} />
                <CopyButton iconOnly className="btn-ghost" text={image.links[fmt]} label={`复制 ${fmtLabel(fmt)}`} />
              </div>
            </div>
          )}

          <dl className="kv">
            {!trash && (
              <KeyValue k="公开">
                <Switch checked={pub} label="公开" onChange={(v) => lib.act.setVis([image.id], v ? 'public' : 'private')} />
              </KeyValue>
            )}
            {!trash && (
              <KeyValue k="标签" top>
                <TagEditor image={image} known={lib.knownTags} onChange={(tags) => lib.act.setTags(image.id, tags)} />
              </KeyValue>
            )}
            <KeyValue k="来源">{sourceLabel(image.source)}</KeyValue>
            <KeyValue k="上传时间">
              <span className="num">{fmtDate(image.createdAt)}</span>
            </KeyValue>
            {trash && (
              <KeyValue k="删除时间">
                <span className="num">{fmtDate(image.trashedAt)}</span>
              </KeyValue>
            )}
            <KeyValue k="ID">
              <span className="mono truncate">{image.id}</span>
              <CopyButton iconOnly size="sm" className="btn-ghost" text={image.id} label="复制 ID" />
            </KeyValue>
            <KeyValue k="SHA-256">
              <span className="mono truncate" title={image.sha256}>
                {image.sha256}
              </span>
              <CopyButton iconOnly size="sm" className="btn-ghost" text={image.sha256} label="复制 SHA-256" />
            </KeyValue>
          </dl>
        </div>
      </div>

      <div className="detail-foot">
        {trash ? (
          <>
            <button type="button" className="btn btn-ghost" onClick={() => lib.act.restore([image.id])}>
              <Icon name="restore" />
              恢复
            </button>
            <button type="button" className="btn btn-danger-ghost" onClick={() => lib.act.purge([image.id])}>
              永久删除
            </button>
          </>
        ) : (
          <>
            <button type="button" className="btn btn-ghost" onClick={() => lib.act.download(image)}>
              <Icon name="download" />
              下载
            </button>
            <button type="button" className="btn btn-danger-ghost" onClick={() => lib.act.trash([image.id])}>
              <Icon name="trash" />
              移到回收站
            </button>
          </>
        )}
      </div>
    </div>
  );
}
// ---------------------------------------------------------------- upload

function Dropzone({ lib }) {
  return (
    <button type="button" className="dropzone" onClick={lib.pick}>
      <span className="dz-icon">
        <Icon name="upload" size={22} />
      </span>
      <span className="dz-main">
        拖放或粘贴图片，或 <span className="dz-link">选择文件</span>
      </span>
    </button>
  );
}

function UploadList({ lib }) {
  return (
    <ul className="up-list">
      {lib.uploads.map((u) => {
        const img = u.imageId ? lib.byId[u.imageId] : null;
        const thumbSrc = { thumb: u.thumb, format: u.format };
        return (
          <li className={`up-row up-${u.status}`} key={u.id}>
            <Thumb image={img ?? thumbSrc} />
            <div className="up-info">
              <span className="name truncate" title={u.name}>
                {u.name}
              </span>
              {u.status === 'uploading' && (
                <>
                  <div className="progress" role="progressbar" aria-valuenow={u.progress} aria-valuemin={0} aria-valuemax={100} aria-label={`${u.name} 上传进度`}>
                    <span style={{ width: `${u.progress}%` }} />
                  </div>
                  <span className="sub num">
                    {u.progress}% · {fmtBytes(Math.round((u.size * u.progress) / 100))} / {fmtBytes(u.size)}
                  </span>
                </>
              )}
              {u.status === 'queued' && <span className="sub">等待上传</span>}
              {u.status === 'processing' && (
                <>
                  <div className="progress indeterminate">
                    <span />
                  </div>
                  <span className="sub">处理中</span>
                </>
              )}
              {u.status === 'done' && (
                <span className="sub ok">
                  <Icon name="check" size={12} />
                  已上传 · {fmtBytes(u.size)}
                  {img?.visibility === 'private' ? ' · 私有' : ''}
                </span>
              )}
              {u.status === 'duplicate' && <span className="sub">已存在相同图片，返回原链接</span>}
              {u.status === 'error' && <span className="sub err">{u.error}</span>}
            </div>
            <div className="up-act">
              {(u.status === 'uploading' || u.status === 'queued' || u.status === 'processing') && (
                <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label={`取消上传 ${u.name}`} title="取消上传" onClick={() => lib.cancelUpload(u.id)}>
                  <Icon name="x" />
                </button>
              )}
              {(u.status === 'done' || u.status === 'duplicate') && img && <CopyButton iconOnly size="sm" className="btn-ghost" text={img.links[lib.linkFormat]} label={`复制 ${fmtLabel(lib.linkFormat)}`} />}
              {u.status === 'error' &&
                (u.retryable ? (
                  <button type="button" className="btn btn-secondary btn-sm" onClick={() => lib.retryUpload(u.id)}>
                    <Icon name="refresh" />
                    重试
                  </button>
                ) : (
                  <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label={`移除 ${u.name}`} title="移除" onClick={() => lib.removeUpload(u.id)}>
                    <Icon name="x" />
                  </button>
                ))}
            </div>
          </li>
        );
      })}
    </ul>
  );
}

function uploadSummary(uploads) {
  const active = countWhere(uploads, (u) => u.status === 'uploading' || u.status === 'queued' || u.status === 'processing');
  const done = countWhere(uploads, (u) => u.status === 'done' || u.status === 'duplicate');
  const failed = countWhere(uploads, (u) => u.status === 'error');
  return { active, done, failed, total: uploads.length };
}

function copyFinished(lib) {
  const imgs = lib.uploads.filter((u) => u.imageId && lib.byId[u.imageId]).map((u) => lib.byId[u.imageId]);
  if (imgs.length) copyMany(lib, imgs);
}

function UploadDialog({ lib, onClose }) {
  const s = uploadSummary(lib.uploads);
  const counted = lib.uploads.filter((u) => u.status !== 'error');
  const pct = counted.length ? Math.round(counted.reduce((n, u) => n + (u.status === 'uploading' ? u.progress : u.status === 'queued' ? 0 : 100), 0) / counted.length) : 0;
  return (
    <Modal
      title="上传图片"
      onClose={onClose}
      width="560px"
      className="full-sm"
      footer={(close) => (
        <>
          <Seg label="上传后可见性" value={lib.uploadVis} onChange={lib.setUploadVis} options={VIS_OPTS} className="foot-start" />
          {s.done > 0 && (
            <button type="button" className="btn btn-ghost" onClick={lib.clearFinished}>
              清除已完成
            </button>
          )}
          {s.done > 1 && (
            <button type="button" className="btn btn-secondary" onClick={() => copyFinished(lib)}>
              <Icon name="copy" />
              复制全部链接
            </button>
          )}
          <button type="button" className="btn btn-primary" onClick={close}>
            {s.active ? '在后台继续' : '完成'}
          </button>
        </>
      )}
    >
      <Dropzone lib={lib} />
      {lib.uploads.length > 0 && (
        <div className="up-summary">
          <div className="up-summary-row">
            <span>
              {s.active ? `正在上传 ${s.active} 个，已完成 ${s.done} 个` : `已完成 ${s.done} 个`}
              {s.failed ? `，失败 ${s.failed} 个` : ''}
            </span>
            {s.active > 0 && <span className="num subtle">{pct}%</span>}
          </div>
          <UploadList lib={lib} />
        </div>
      )}
    </Modal>
  );
}

function PurgeConfirm({ count, onCancel, onConfirm }) {
  return (
    <Modal
      title={count === 1 ? '永久删除这张图片？' : `永久删除 ${count} 张图片？`}
      onClose={onCancel}
      width="420px"
      footer={(close) => (
        <>
          <button type="button" className="btn btn-secondary" onClick={close} autoFocus>
            取消
          </button>
          <button type="button" className="btn btn-danger-solid" onClick={onConfirm}>
            永久删除
          </button>
        </>
      )}
    >
      <p>文件会立即从磁盘删除，已发出的链接将失效。此操作无法撤销。</p>
    </Modal>
  );
}

function DropOverlay() {
  return (
    <div className="drop-overlay" aria-hidden="true">
      <div>
        <span className="dz-icon">
          <Icon name="upload" size={22} />
        </span>
        <strong>松开以上传</strong>
      </div>
    </div>
  );
}

Object.assign(window, {
  VIS_OPTS, countWhere, fmtLabel, FilterChips, AdjustSheet, LibraryHead, BulkBar, ImageCollection, DetailPanel, UploadDialog, PurgeConfirm, DropOverlay, copyMany,
});
