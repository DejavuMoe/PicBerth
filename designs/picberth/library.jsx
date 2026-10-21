// Image management building blocks shared by the three directions.
// Every component receives `lib`, the library controller created in app.jsx.

const FORMATS = ['jpeg', 'png', 'gif', 'webp', 'avif'];
const VIS_OPTS = [
  { key: 'public', label: '公开' },
  { key: 'private', label: '私有' },
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
function hasAlpha(im) {
  return im.format === 'png' || im.format === 'webp' || im.format === 'gif';
}
function stop(e) {
  e.stopPropagation();
}

function RadioItem({ on, children, end, onClick }) {
  return (
    <button type="button" role="menuitemradio" aria-checked={on} className="menu-item" onClick={onClick}>
      <span className="menu-tick">{on && <Icon name="check" />}</span>
      <span className="truncate">{children}</span>
      {end !== undefined && <span className="end">{end}</span>}
    </button>
  );
}

// ---------------------------------------------------------------- filters

function FilterButton({ label, n, open, toggle }) {
  return (
    <button type="button" className={`btn btn-secondary filter-btn ${n ? 'active' : ''}`} aria-expanded={open} onClick={toggle}>
      <span>{label}</span>
      {n ? <span className="filter-n">{n}</span> : null}
      <Icon name="chevron-down" size={14} />
    </button>
  );
}

function FilterMenus({ lib }) {
  const f = lib.filters;
  const base = lib.base;
  const set = (p) => lib.setFilters({ ...f, ...p });
  return (
    <>
      <Menu trigger={(open, toggle) => <FilterButton label="格式" n={f.formats.length} open={open} toggle={toggle} />}>
        {FORMATS.map((k) => (
          <MenuItem key={k} checked={f.formats.includes(k)} end={countWhere(base, (i) => i.format === k)} onClick={() => set({ formats: toggleIn(f.formats, k) })}>
            {FORMAT_LABEL[k]}
          </MenuItem>
        ))}
      </Menu>
      <Menu trigger={(open, toggle) => <FilterButton label="可见性" n={f.vis ? 1 : 0} open={open} toggle={toggle} />}>
        {(close) => (
          <>
            <RadioItem on={!f.vis} onClick={() => (set({ vis: '' }), close())} end={base.length}>
              全部
            </RadioItem>
            {VIS_OPTS.map((o) => (
              <RadioItem key={o.key} on={f.vis === o.key} end={countWhere(base, (i) => i.visibility === o.key)} onClick={() => (set({ vis: o.key }), close())}>
                {o.label}
              </RadioItem>
            ))}
          </>
        )}
      </Menu>
      {lib.ext && (
        <Menu trigger={(open, toggle) => <FilterButton label="上传时间" n={f.date ? 1 : 0} open={open} toggle={toggle} />}>
          {(close) => (
            <>
              <RadioItem on={!f.date} onClick={() => (set({ date: '' }), close())}>
                不限
              </RadioItem>
              {DATE_RANGES.map((d) => (
                <RadioItem key={d.key} on={f.date === d.key} onClick={() => (set({ date: d.key }), close())}>
                  {d.label}
                </RadioItem>
              ))}
            </>
          )}
        </Menu>
      )}
      {lib.ext && (
        <Menu width={220} trigger={(open, toggle) => <FilterButton label="来源" n={f.sources.length} open={open} toggle={toggle} />}>
          {SOURCES.map((s) => (
            <MenuItem key={s.key} checked={f.sources.includes(s.key)} end={countWhere(base, (i) => i.source === s.key)} onClick={() => set({ sources: toggleIn(f.sources, s.key) })}>
              {s.label}
            </MenuItem>
          ))}
        </Menu>
      )}
      {lib.ext && (
        <Menu trigger={(open, toggle) => <FilterButton label="标签" n={f.tags.length} open={open} toggle={toggle} />}>
          {allTags(base).map(([t, n]) => (
            <MenuItem key={t} checked={f.tags.includes(t)} end={n} onClick={() => set({ tags: toggleIn(f.tags, t) })}>
              {t}
            </MenuItem>
          ))}
        </Menu>
      )}
      <button type="button" className={`btn btn-secondary filter-btn ${f.gps ? 'active' : ''}`} aria-pressed={f.gps} onClick={() => set({ gps: !f.gps })}>
        <Icon name="pin" />
        <span>含位置</span>
      </button>
    </>
  );
}

function FilterChips({ lib }) {
  const f = lib.filters;
  const set = (p) => lib.setFilters({ ...f, ...p });
  const chips = [];
  f.formats.forEach((k) => chips.push({ k: 'f' + k, label: FORMAT_LABEL[k], rm: () => set({ formats: f.formats.filter((x) => x !== k) }) }));
  if (f.vis) chips.push({ k: 'vis', label: f.vis === 'public' ? '公开' : '私有', rm: () => set({ vis: '' }) });
  if (f.gps) chips.push({ k: 'gps', label: '含位置', rm: () => set({ gps: false }) });
  if (lib.ext) {
    if (f.date) chips.push({ k: 'date', label: DATE_RANGES.find((d) => d.key === f.date).label, rm: () => set({ date: '' }) });
    f.sources.forEach((s) => chips.push({ k: 's' + s, label: sourceLabel(s), rm: () => set({ sources: f.sources.filter((x) => x !== s) }) }));
    f.tags.forEach((t) => chips.push({ k: 't' + t, label: `#${t}`, rm: () => set({ tags: f.tags.filter((x) => x !== t) }) }));
  }
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

/** Faceted filter list: left column in direction B, filter sheet on phones. */
function FacetPanel({ lib }) {
  const f = lib.filters;
  const base = lib.base;
  const set = (p) => lib.setFilters({ ...f, ...p });
  const opt = (key, on, label, count, onClick, multi) => (
    <button type="button" key={key} className={`facet-opt ${on ? 'on' : ''}`} aria-pressed={on} onClick={onClick}>
      {multi ? <input type="checkbox" className="check" checked={on} readOnly tabIndex={-1} aria-hidden="true" /> : <span className={`radio ${on ? 'on' : ''}`} aria-hidden="true" />}
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
    <div className="facet-list">
      {lib.ext && (
        <div className="facet">
          {head('上传时间', !!f.date, () => set({ date: '' }))}
          {opt('all', !f.date, '不限', base.length, () => set({ date: '' }))}
          {DATE_RANGES.map((d) =>
            opt(d.key, f.date === d.key, d.label, countWhere(base, (i) => (d.since ? i.createdAt >= d.since() : i.createdAt < d.until())), () => set({ date: f.date === d.key ? '' : d.key })),
          )}
        </div>
      )}
      <div className="facet">
        {head('可见性', !!f.vis, () => set({ vis: '' }))}
        {opt('all', !f.vis, '全部', base.length, () => set({ vis: '' }))}
        {VIS_OPTS.map((o) => opt(o.key, f.vis === o.key, o.label, countWhere(base, (i) => i.visibility === o.key), () => set({ vis: f.vis === o.key ? '' : o.key })))}
      </div>
      <div className="facet">
        {head('格式', f.formats.length > 0, () => set({ formats: [] }))}
        {FORMATS.map((k) => {
          const n = countWhere(base, (i) => i.format === k);
          return opt(k, f.formats.includes(k), FORMAT_LABEL[k], n, () => set({ formats: toggleIn(f.formats, k) }), true);
        })}
      </div>
      {lib.ext && (
        <div className="facet">
          {head('来源', f.sources.length > 0, () => set({ sources: [] }))}
          {SOURCES.map((s) => opt(s.key, f.sources.includes(s.key), s.label, countWhere(base, (i) => i.source === s.key), () => set({ sources: toggleIn(f.sources, s.key) }), true))}
        </div>
      )}
      {lib.ext && (
        <div className="facet">
          {head('标签', f.tags.length > 0, () => set({ tags: [] }))}
          {allTags(base).map(([t, n]) => opt(t, f.tags.includes(t), t, n, () => set({ tags: toggleIn(f.tags, t) }), true))}
        </div>
      )}
      <div className="facet">
        {head('位置信息', f.gps, () => set({ gps: false }))}
        {opt('gps', f.gps, '含位置', countWhere(base, (i) => i.hasGps), () => set({ gps: !f.gps }), true)}
      </div>
    </div>
  );
}

/** Phone filter sheet; same facets, applied live. */
function FilterSheet({ lib, onClose }) {
  const n = activeFilterCount({ ...lib.filters, q: '' }, lib.ext);
  return (
    <Sheet onClose={onClose} label="筛选">
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
          {SORTS.filter((s) => lib.ext || s.key !== 'name').map((s) => (
            <button type="button" key={s.key} className={`facet-opt ${lib.sort === s.key ? 'on' : ''}`} aria-pressed={lib.sort === s.key} onClick={() => lib.setSort(s.key)}>
              <span className={`radio ${lib.sort === s.key ? 'on' : ''}`} aria-hidden="true" />
              {s.label}
            </button>
          ))}
        </div>
        <FacetPanel lib={lib} />
      </div>
      <div className="sheet-foot">
        <button type="button" className="btn btn-primary btn-block" onClick={onClose}>
          显示 {lib.items.length} 张图片
        </button>
      </div>
    </Sheet>
  );
}

function SortMenu({ lib, iconOnly }) {
  const opts = SORTS.filter((s) => lib.ext || s.key !== 'name');
  const cur = opts.find((s) => s.key === lib.sort) ?? opts[0];
  return (
    <Menu
      align="right"
      width={160}
      trigger={(open, toggle) => (
        <button type="button" className={`btn btn-secondary ${iconOnly ? 'btn-icon' : ''}`} aria-expanded={open} aria-label={iconOnly ? `排序：${cur.label}` : undefined} title={iconOnly ? `排序：${cur.label}` : undefined} onClick={toggle}>
          <Icon name="sort" />
          {!iconOnly && <span>{cur.label}</span>}
        </button>
      )}
    >
      {(close) =>
        opts.map((s) => (
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

function SearchBox({ lib, inputRef, placeholder = '搜索文件名、ID 或 SHA-256', size }) {
  return (
    <label className="search">
      <Icon name="search" />
      <input
        ref={inputRef}
        className="input"
        type="search"
        placeholder={placeholder}
        aria-label="搜索图片"
        value={lib.filters.q}
        onChange={(e) => lib.setFilters({ ...lib.filters, q: e.target.value })}
        onKeyDown={(e) => e.key === 'Escape' && lib.setFilters({ ...lib.filters, q: '' })}
        style={size ? { height: size } : undefined}
      />
      {!lib.filters.q && <kbd className="hide-sm">/</kbd>}
    </label>
  );
}

/** Toolbar used by directions A and C: search, filter menus, sort, view. */
function LibraryToolbar({ lib, searchRef, onOpenFilters }) {
  const n = activeFilterCount({ ...lib.filters, q: '' }, lib.ext);
  if (lib.sel.size || lib.selectMode) return <BulkBar lib={lib} />;
  if (lib.dataState === 'ok' && !lib.base.length) return null;
  return (
    <>
      <div className="toolbar">
        <SearchBox lib={lib} inputRef={searchRef} />
        {lib.isMobile ? (
          <>
            <button type="button" className={`btn btn-secondary filter-btn ${n ? 'active' : ''}`} onClick={onOpenFilters}>
              <Icon name="filter" />
              <span>筛选</span>
              {n ? <span className="filter-n">{n}</span> : null}
            </button>
            <span className="grow" />
            <button type="button" className="btn btn-ghost" onClick={() => lib.setSelectMode(true)}>
              选择
            </button>
            <ViewToggle lib={lib} />
          </>
        ) : (
          <>
            <FilterMenus lib={lib} />
            <span className="grow" />
            <SortMenu lib={lib} />
            <ViewToggle lib={lib} />
          </>
        )}
      </div>
      <FilterChips lib={lib} />
    </>
  );
}

// ---------------------------------------------------------------- selection

function copyMany(lib, imgs) {
  writeClipboard(imgs.map((i) => i.links[lib.linkFormat]).join('\n'));
  lib.toast(imgs.length === 1 ? `已复制 ${fmtLabel(lib.linkFormat)} 链接` : `已复制 ${imgs.length} 条 ${fmtLabel(lib.linkFormat)} 链接`);
}

function BulkBar({ lib }) {
  const ids = [...lib.sel];
  const imgs = ids.map((id) => lib.byId[id]).filter(Boolean);
  const trash = lib.route === 'trash';
  const all = lib.items.length;
  const m = lib.isMobile;
  if (!ids.length)
    return (
      <div className="bulkbar" role="toolbar" aria-label="批量操作">
        <span className="count">点按图片以选择</span>
        <span className="grow" />
        <button type="button" className="btn btn-ghost btn-sm" onClick={lib.selectAll}>
          全选
        </button>
        <button type="button" className="btn btn-ghost btn-sm" onClick={lib.clearSel}>
          完成
        </button>
      </div>
    );
  return (
    <div className="bulkbar" role="toolbar" aria-label="批量操作">
      <span className="count">已选 {ids.length} 张</span>
      {ids.length < all && (
        <button type="button" className="btn btn-ghost btn-sm" onClick={lib.selectAll}>
          {m ? '全选' : `全选 ${all} 张`}
        </button>
      )}
      <span className="grow" />
      {trash ? (
        <>
          <button type="button" className="btn btn-secondary btn-sm" onClick={() => lib.act.restore(ids)}>
            <Icon name="restore" />
            恢复
          </button>
          <button type="button" className="btn btn-danger btn-sm" onClick={() => lib.act.purge(ids)}>
            永久删除
          </button>
        </>
      ) : (
        <>
          <button type="button" className={`btn btn-secondary btn-sm ${m ? 'btn-icon' : ''}`} aria-label={m ? '复制链接' : undefined} onClick={() => copyMany(lib, imgs)}>
            <Icon name="copy" />
            {!m && '复制链接'}
          </button>
          {m ? (
            <Menu
              align="right"
              up
              trigger={(open, toggle) => (
                <button type="button" className="btn btn-secondary btn-sm btn-icon" aria-label="可见性" aria-expanded={open} onClick={toggle}>
                  <Icon name="lock" />
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
          ) : (
            <>
              <button type="button" className="btn btn-secondary btn-sm" onClick={() => lib.act.setVis(ids, 'public')}>
                <Icon name="globe" />
                设为公开
              </button>
              <button type="button" className="btn btn-secondary btn-sm" onClick={() => lib.act.setVis(ids, 'private')}>
                <Icon name="lock" />
                设为私有
              </button>
            </>
          )}
          <button type="button" className={`btn btn-danger btn-sm ${m ? 'btn-icon' : ''}`} aria-label={m ? '移到回收站' : undefined} onClick={() => lib.act.trash(ids)}>
            <Icon name="trash" />
            {!m && '移到回收站'}
          </button>
        </>
      )}
      <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label="取消选择" title="取消选择 (Esc)" onClick={lib.clearSel}>
        <Icon name="x" />
      </button>
    </div>
  );
}

// ---------------------------------------------------------------- per-image menu

function RowMenu({ lib, image, up }) {
  const trash = image.state === 'trashed';
  return (
    <span onClick={stop}>
      <Menu
        align="right"
        up={up}
        width={180}
        trigger={(open, toggle) => (
          <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label={`${image.filename} 的更多操作`} aria-expanded={open} onClick={toggle}>
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
              <MenuItem icon="image" onClick={run(() => lib.open(image.id))}>
                查看详情
              </MenuItem>
              <MenuItem icon="external" onClick={run(() => lib.toast('已在新标签页打开原图'))}>
                打开原图
              </MenuItem>
              <MenuItem icon="download" onClick={run(() => lib.act.download(image))}>
                下载
              </MenuItem>
              <MenuItem icon="clipboard" onClick={run(() => (writeClipboard(image.id), lib.toast('已复制 ID')))}>
                复制 ID
              </MenuItem>
              <div className="menu-sep" />
              {image.visibility === 'public' ? (
                <MenuItem icon="lock" onClick={run(() => lib.act.setVis([image.id], 'private'))}>
                  设为私有
                </MenuItem>
              ) : (
                <MenuItem icon="globe" onClick={run(() => lib.act.setVis([image.id], 'public'))}>
                  设为公开
                </MenuItem>
              )}
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

function NameSub({ lib, image }) {
  return (
    <span className="sub">
      <span>{FORMAT_LABEL[image.format]}</span>
      {image.frames > 1 && <span>· 动图</span>}
      {image.hasGps && (
        <span className="gps" title="含位置信息">
          · <Icon name="pin" size={12} /> 含位置
        </span>
      )}
      {lib.ext &&
        image.tags.map((t) => (
          <span key={t} className="tag">
            {t}
          </span>
        ))}
    </span>
  );
}

// ---------------------------------------------------------------- table (desktop)

function SortHead({ lib, label, desc, asc, numeric }) {
  const on = lib.sort === desc || lib.sort === asc;
  const dir = lib.sort === asc ? 'ascending' : 'descending';
  return (
    <th className={numeric ? 'c-num' : undefined} aria-sort={on ? dir : 'none'}>
      <button type="button" className={on ? 'sorted' : ''} onClick={() => lib.setSort(lib.sort === desc ? asc : desc)}>
        {label}
        <Icon name={on && lib.sort === asc ? 'chevron-up' : 'chevron-down'} size={12} />
      </button>
    </th>
  );
}

function ImageTable({ lib, items, compact }) {
  const trash = lib.route === 'trash';
  const allOn = items.length > 0 && items.every((i) => lib.sel.has(i.id));
  const someOn = !allOn && items.some((i) => lib.sel.has(i.id));
  const headRef = React.useRef(null);
  React.useEffect(() => {
    if (headRef.current) headRef.current.indeterminate = someOn;
  }, [someOn]);
  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th className="c-check">
              <input ref={headRef} type="checkbox" className="check" checked={allOn} onChange={() => (allOn ? lib.clearSel() : lib.selectAll())} aria-label="全选" />
            </th>
            <th className="c-thumb">
              <span className="sr-only">预览</span>
            </th>
            {lib.ext ? <SortHead lib={lib} label="文件名" desc="name" asc="name" /> : <th>文件名</th>}
            {!compact && <th className="c-num">尺寸</th>}
            <SortHead lib={lib} label="大小" desc="large" asc="small" numeric />
            {trash ? (
              <>
                <th>删除时间</th>
                <th>自动清除</th>
              </>
            ) : (
              <>
                <th>可见性</th>
                <SortHead lib={lib} label="上传时间" desc="new" asc="old" />
              </>
            )}
            <th className="c-actions">
              <span className="sr-only">操作</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {items.map((im, idx) => {
            const on = lib.sel.has(im.id);
            return (
              <tr
                key={im.id}
                className={`${on ? 'selected' : ''} ${lib.activeId === im.id ? 'active' : ''}`}
                onClick={(e) => lib.clickItem(im.id, e)}
                tabIndex={0}
                onKeyDown={(e) => {
                  if (e.target !== e.currentTarget) return;
                  if (e.key === 'Enter') lib.open(im.id);
                  if (e.key === ' ') {
                    e.preventDefault();
                    lib.toggle(im.id);
                  }
                }}
                aria-selected={on}
              >
                <td className="c-check" onClick={stop}>
                  <input type="checkbox" className="check" checked={on} onChange={() => {}} onClick={(e) => lib.toggle(im.id, e.shiftKey)} aria-label={`选择 ${im.filename}`} />
                </td>
                <td className="c-thumb">
                  <Thumb image={im} className="thumb-sm" />
                </td>
                <td>
                  <span className="cell-name">
                    <span className="name truncate" title={im.filename}>
                      {im.filename}
                    </span>
                    <NameSub lib={lib} image={im} />
                  </span>
                </td>
                {!compact && (
                  <td className="c-num subtle">
                    {im.width} × {im.height}
                  </td>
                )}
                <td className="c-num">{fmtBytes(im.size)}</td>
                {trash ? (
                  <>
                    <td className="nowrap subtle" title={fmtDate(im.trashedAt)}>
                      {fmtRel(im.trashedAt)}
                    </td>
                    <td className="nowrap">
                      <span className="badge">{daysLeft(im, lib.settings.trashRetentionDays)}</span>
                    </td>
                  </>
                ) : (
                  <>
                    <td>
                      <VisBadge v={im.visibility} />
                    </td>
                    <td className="nowrap subtle" title={fmtDate(im.createdAt)}>
                      {fmtRel(im.createdAt)}
                    </td>
                  </>
                )}
                <td className="c-actions">
                  <span className="cell-actions">
                    {trash ? (
                      <button
                        type="button"
                        className="btn btn-secondary btn-sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          lib.act.restore([im.id]);
                        }}
                      >
                        恢复
                      </button>
                    ) : (
                      <CopyLinkSplit image={im} format={lib.linkFormat} onFormat={lib.setLinkFormat} />
                    )}
                    <RowMenu lib={lib} image={im} up={items.length > 5 && idx >= items.length - 3} />
                  </span>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
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
      <Thumb image={image} cover />
      <label className="tile-check" onClick={stop}>
        <input type="checkbox" className="check" checked={on} onChange={() => {}} onClick={(e) => lib.toggle(image.id, e.shiftKey)} aria-label={`选择 ${image.filename}`} />
      </label>
      <span className="tile-badges">
        {image.visibility === 'private' && (
          <span className="badge">
            <Icon name="lock" size={12} />
            私有
          </span>
        )}
        {image.frames > 1 && <span className="badge">GIF</span>}
        {image.hasGps && (
          <span className="badge" title="含位置信息" aria-label="含位置信息">
            <Icon name="pin" size={12} />
          </span>
        )}
      </span>
      {!trash && (
        <span className="tile-copy" onClick={stop}>
          <CopyButton iconOnly size="sm" text={image.links[lib.linkFormat]} label={`复制 ${fmtLabel(lib.linkFormat)}`} />
        </span>
      )}
      <span className="tile-meta">
        <span className="name truncate" title={image.filename}>
          {image.filename}
        </span>
        <span className="sub truncate">{trash ? `${fmtBytes(image.size)} · ${daysLeft(image, lib.settings.trashRetentionDays)}` : `${FORMAT_LABEL[image.format]} · ${fmtBytes(image.size)} · ${fmtRel(image.createdAt)}`}</span>
      </span>
    </div>
  );
}

function ImageGrid({ lib, items, groupDays, tile }) {
  const selecting = lib.sel.size > 0 || lib.selectMode;
  const body = (list) => (
    <div className={`grid ${selecting ? 'selecting' : ''}`} style={tile ? { '--tile': tile } : undefined}>
      {list.map((im) => (
        <Tile key={im.id} lib={lib} image={im} />
      ))}
    </div>
  );
  if (!groupDays) return body(items);
  return groupByDay(items).map((g) => (
    <section key={g.key} aria-label={g.label[0]}>
      <h2 className="date-group">
        {g.label[0]}
        <span>
          {g.label[1]} · {g.items.length} 张
        </span>
      </h2>
      {body(g.items)}
    </section>
  ));
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
      }, 450);
    },
    onPointerUp: cancel,
    onPointerLeave: cancel,
    onPointerMove: cancel,
  };
}

function ImageRows({ lib, items }) {
  const selecting = lib.sel.size > 0 || lib.selectMode;
  const trash = lib.route === 'trash';
  return (
    <div className="rows">
      {items.map((im) => {
        const on = lib.sel.has(im.id);
        return (
          <div key={im.id} className={`row ${on ? 'selected' : ''}`} {...longPress(lib, im.id)}>
            {selecting && <input type="checkbox" className="check" checked={on} onChange={() => lib.toggle(im.id)} aria-label={`选择 ${im.filename}`} />}
            <button type="button" className="row-btn" onClick={(e) => lib.clickItem(im.id, e)}>
              <Thumb image={im} cover />
              <span className="row-main">
                <span className="name truncate">{im.filename}</span>
                <span className="sub truncate">
                  {trash
                    ? `${fmtBytes(im.size)} · ${daysLeft(im, lib.settings.trashRetentionDays)}`
                    : `${fmtBytes(im.size)} · ${fmtRel(im.createdAt)}${im.visibility === 'private' ? ' · 私有' : ''}${im.hasGps ? ' · 含位置' : ''}`}
                </span>
              </span>
            </button>
            {!selecting && (trash ? <RowMenu lib={lib} image={im} /> : <CopyButton iconOnly className="btn-ghost" text={im.links[lib.linkFormat]} label={`复制 ${fmtLabel(lib.linkFormat)}`} />)}
          </div>
        );
      })}
    </div>
  );
}

// ---------------------------------------------------------------- collection with states

function LoadingList({ grid }) {
  if (grid)
    return (
      <div className="grid" aria-busy="true" aria-label="正在加载">
        {Array.from({ length: 12 }, (_, i) => (
          <div key={i} className="tile" style={{ cursor: 'default' }}>
            <div className="skeleton" style={{ aspectRatio: '4 / 3', borderRadius: 0 }} />
            <div className="tile-meta">
              <div className="skeleton" style={{ height: 12, width: '70%', margin: '3px 0' }} />
              <div className="skeleton" style={{ height: 10, width: '45%', margin: '3px 0' }} />
            </div>
          </div>
        ))}
      </div>
    );
  return (
    <div className="rows" aria-busy="true" aria-label="正在加载">
      {Array.from({ length: 8 }, (_, i) => (
        <div key={i} className="row">
          <div className="skeleton" style={{ width: 40, height: 40, flex: 'none' }} />
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 6 }}>
            <div className="skeleton" style={{ height: 12, width: `${40 + ((i * 17) % 35)}%` }} />
            <div className="skeleton" style={{ height: 10, width: '22%' }} />
          </div>
        </div>
      ))}
    </div>
  );
}

function EmptyState({ lib }) {
  if (lib.route === 'trash')
    return (
      <div className="empty">
        <Icon name="trash" size={28} />
        <strong>回收站是空的</strong>
        <p>移到回收站的图片会保留 {lib.settings.trashRetentionDays} 天，之后自动永久删除。</p>
      </div>
    );
  return (
    <div className="empty">
      <Icon name="image" size={28} />
      <strong>还没有图片</strong>
      <p>把图片拖到此页面、直接粘贴，或选择文件上传。</p>
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

function ImageCollection({ lib, groupDays, tile, compact }) {
  const grid = lib.view === 'grid';
  if (lib.dataState === 'loading') return <LoadingList grid={grid} />;
  if (lib.dataState === 'error')
    return (
      <div className="alert danger" role="alert">
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
        <Icon name="search" size={28} />
        <strong>没有符合条件的图片</strong>
        <p>修改搜索词或清除筛选条件后再试。</p>
        <div className="actions">
          <button type="button" className="btn btn-secondary" onClick={() => lib.setFilters(EMPTY_FILTERS)}>
            清除搜索和筛选
          </button>
        </div>
      </div>
    );
  const total = lib.items.reduce((s, i) => s + i.size, 0);
  return (
    <>
      {grid ? <ImageGrid lib={lib} items={lib.items} groupDays={groupDays} tile={tile} /> : lib.isMobile ? <ImageRows lib={lib} items={lib.items} /> : <ImageTable lib={lib} items={lib.items} compact={compact} />}
      <div className="pager">
        <span>
          {lib.items.length < lib.base.length ? `${lib.items.length} / ${lib.base.length} 张` : `共 ${lib.items.length} 张`} · {fmtBytes(total)}
        </span>
        <span>已全部加载</span>
      </div>
    </>
  );
}

// ---------------------------------------------------------------- detail

function RenameField({ image, onSave }) {
  const [edit, setEdit] = React.useState(false);
  const [v, setV] = React.useState(image.filename);
  React.useEffect(() => {
    setEdit(false);
    setV(image.filename);
  }, [image.id, image.filename]);
  const err = !v.trim() ? '请输入文件名' : v.length > 200 ? '文件名不能超过 200 个字符' : null;
  if (!edit)
    return (
      <div className="setting-row">
        <span className="truncate" title={image.filename}>
          {image.filename}
        </span>
        <button type="button" className="btn btn-ghost btn-sm" onClick={() => setEdit(true)}>
          <Icon name="edit" />
          重命名
        </button>
      </div>
    );
  const save = () => {
    if (err) return;
    onSave(v.trim());
    setEdit(false);
  };
  return (
    <>
      <div className="rename">
        <input
          className="input"
          autoFocus
          value={v}
          aria-label="文件名"
          aria-invalid={!!err}
          onChange={(e) => setV(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') save();
            if (e.key === 'Escape') {
              e.stopPropagation();
              setEdit(false);
              setV(image.filename);
            }
          }}
        />
        <button type="button" className="btn btn-primary" disabled={!!err} onClick={save}>
          保存
        </button>
        <button type="button" className="btn btn-ghost" onClick={() => (setEdit(false), setV(image.filename))}>
          取消
        </button>
      </div>
      {err ? <p className="error-text" style={{ marginTop: 6 }}>{err}</p> : <p className="hint" style={{ marginTop: 6 }}>只修改显示的文件名，链接不变。</p>}
    </>
  );
}

function TagEditor({ image, onChange, known }) {
  const [v, setV] = React.useState('');
  const add = () => {
    const t = v.trim();
    if (t && !image.tags.includes(t)) onChange([...image.tags, t]);
    setV('');
  };
  const listId = `tags-${image.id}`;
  return (
    <div className="tag-editor">
      {image.tags.map((t) => (
        <span key={t} className="tag">
          {t}
          <button type="button" aria-label={`移除标签 ${t}`} onClick={() => onChange(image.tags.filter((x) => x !== t))}>
            <Icon name="x" size={12} />
          </button>
        </span>
      ))}
      <input
        className="input"
        placeholder="添加标签"
        aria-label="添加标签"
        list={listId}
        value={v}
        maxLength={20}
        onChange={(e) => setV(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ',') {
            e.preventDefault();
            add();
          }
          if (e.key === 'Backspace' && !v && image.tags.length) onChange(image.tags.slice(0, -1));
        }}
        onBlur={add}
      />
      <datalist id={listId}>
        {known
          .filter((t) => !image.tags.includes(t))
          .map((t) => (
            <option key={t} value={t} />
          ))}
      </datalist>
    </div>
  );
}

function LinkSection({ lib, image }) {
  const fmt = lib.linkFormat;
  return (
    <section>
      <div className="section-label">链接</div>
      <Seg label="链接格式" value={fmt} onChange={lib.setLinkFormat} options={LINK_FORMATS} className="fill" />
      <div className="link-box">
        <input className="input" readOnly value={image.links[fmt]} aria-label={`${fmtLabel(fmt)} 链接`} onFocus={(e) => e.target.select()} />
        <CopyButton text={image.links[fmt]} className="btn-primary" />
      </div>
      {image.visibility === 'private' && <p className="hint" style={{ marginTop: 6 }}>私有图片的链接只在登录后可以打开。</p>}
    </section>
  );
}

function DetailPanel({ lib, image, onClose, wide, closeIcon = 'x' }) {
  const trash = image.state === 'trashed';
  const [prev, next] = lib.neighbors(image.id);
  const pub = image.visibility === 'public';
  const props = (
    <dl className="props">
      <dt>格式</dt>
      <dd>
        {FORMAT_LABEL[image.format]}
        {image.frames > 1 && <span className="badge">动图 · {image.frames} 帧</span>}
      </dd>
      <dt>尺寸</dt>
      <dd className="num">
        {image.width} × {image.height}
      </dd>
      <dt>大小</dt>
      <dd>{fmtBytes(image.size)}</dd>
      <dt>位置信息</dt>
      <dd>
        {image.hasGps ? (
          <span className="badge badge-warning">
            <Icon name="pin" size={12} />
            含位置
          </span>
        ) : (
          '无'
        )}
      </dd>
      <dt>来源</dt>
      <dd>{sourceLabel(image.source)}</dd>
      <dt>上传时间</dt>
      <dd className="num">{fmtDate(image.createdAt)}</dd>
      {trash && (
        <>
          <dt>删除时间</dt>
          <dd className="num">{fmtDate(image.trashedAt)}</dd>
          <dt>自动清除</dt>
          <dd>{daysLeft(image, lib.settings.trashRetentionDays)}</dd>
        </>
      )}
      <dt>ID</dt>
      <dd>
        <span className="mono truncate">{image.id}</span>
        <CopyButton iconOnly size="sm" className="btn-ghost" text={image.id} label="复制 ID" />
      </dd>
      <dt>SHA-256</dt>
      <dd>
        <span className="mono truncate" title={image.sha256}>
          {image.sha256}
        </span>
        <CopyButton iconOnly size="sm" className="btn-ghost" text={image.sha256} label="复制 SHA-256" />
      </dd>
    </dl>
  );
  const info = (
    <>
      {!trash && <LinkSection lib={lib} image={image} />}
      {lib.ext && !trash && (
        <section>
          <div className="section-label">文件名</div>
          <RenameField image={image} onSave={(n) => lib.act.rename(image.id, n)} />
        </section>
      )}
      {!trash && (
        <section>
          <div className="setting-row">
            <div>
              <div className="label">公开访问</div>
              <div className="hint">{pub ? '任何人都可以通过链接打开' : '只有登录后才能打开'}</div>
            </div>
            <Switch checked={pub} label="公开访问" onChange={(v) => lib.act.setVis([image.id], v ? 'public' : 'private')} />
          </div>
        </section>
      )}
      {lib.ext && !trash && (
        <section>
          <div className="section-label">标签</div>
          <TagEditor image={image} known={lib.knownTags} onChange={(tags) => lib.act.setTags(image.id, tags)} />
        </section>
      )}
      <section>
        <div className="section-label">信息</div>
        {props}
      </section>
    </>
  );
  const preview = (
    <a
      className={`preview ${hasAlpha(image) ? 'checker' : ''}`}
      href={image.url}
      target="_blank"
      rel="noreferrer"
      title="打开原图"
      onClick={(e) => {
        e.preventDefault();
        lib.toast('已在新标签页打开原图');
      }}
    >
      <img src={image.thumb} alt={image.filename} />
    </a>
  );
  return (
    <div className={`detail ${wide ? 'detail-wide' : ''}`}>
      <div className="detail-head">
        <h2 className="truncate" title={image.filename}>
          {image.filename}
        </h2>
        <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label="上一张" title="上一张 (←)" disabled={!prev} onClick={() => prev && lib.open(prev.id)}>
          <Icon name="chevron-left" />
        </button>
        <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label="下一张" title="下一张 (→)" disabled={!next} onClick={() => next && lib.open(next.id)}>
          <Icon name="chevron-right" />
        </button>
        {onClose && (
          <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label="关闭" title="关闭 (Esc)" onClick={onClose}>
            <Icon name={closeIcon} />
          </button>
        )}
      </div>
      {wide ? (
        <div className="detail-body">
          <div className="detail-stage">{preview}</div>
          <div className="detail-scroll">{info}</div>
        </div>
      ) : (
        <div className="detail-scroll">
          {preview}
          {info}
        </div>
      )}
      <div className="detail-foot">
        {trash ? (
          <>
            <button type="button" className="btn btn-secondary" onClick={() => lib.act.restore([image.id])}>
              <Icon name="restore" />
              恢复
            </button>
            <span className="grow" />
            <button type="button" className="btn btn-danger" onClick={() => lib.act.purge([image.id])}>
              永久删除
            </button>
          </>
        ) : (
          <>
            <button type="button" className="btn btn-secondary" onClick={() => lib.act.download(image)}>
              <Icon name="download" />
              下载
            </button>
            <span className="grow" />
            <button type="button" className="btn btn-danger" onClick={() => lib.act.trash([image.id])}>
              <Icon name="trash" />
              移到回收站
            </button>
          </>
        )}
      </div>
    </div>
  );
}

/** Inspector content for a multi-selection (direction B). */
function BatchPanel({ lib }) {
  const imgs = [...lib.sel].map((id) => lib.byId[id]).filter(Boolean);
  const trash = lib.route === 'trash';
  const size = imgs.reduce((s, i) => s + i.size, 0);
  const pub = countWhere(imgs, (i) => i.visibility === 'public');
  const ids = imgs.map((i) => i.id);
  return (
    <div className="detail">
      <div className="detail-head">
        <h2>已选 {imgs.length} 张</h2>
        <button type="button" className="btn btn-ghost btn-sm" onClick={lib.clearSel}>
          取消选择
        </button>
      </div>
      <div className="detail-scroll">
        <div className="mosaic">
          {imgs.slice(0, 8).map((im) => (
            <Thumb key={im.id} image={im} cover />
          ))}
          {imgs.length > 8 && <span className="mosaic-more">+{imgs.length - 8}</span>}
        </div>
        <dl className="props">
          <dt>合计大小</dt>
          <dd>{fmtBytes(size)}</dd>
          {!trash && (
            <>
              <dt>可见性</dt>
              <dd>
                公开 {pub} · 私有 {imgs.length - pub}
              </dd>
            </>
          )}
          <dt>含位置</dt>
          <dd>{countWhere(imgs, (i) => i.hasGps)} 张</dd>
        </dl>
        {!trash && (
          <section>
            <div className="section-label">复制全部链接</div>
            <Seg label="链接格式" value={lib.linkFormat} onChange={lib.setLinkFormat} options={LINK_FORMATS} className="fill" />
            <button type="button" className="btn btn-primary btn-block" style={{ marginTop: 8 }} onClick={() => copyMany(lib, imgs)}>
              <Icon name="copy" />
              复制 {imgs.length} 条链接
            </button>
          </section>
        )}
        {!trash && (
          <section>
            <div className="section-label">可见性</div>
            <div className="btn-row">
              <button type="button" className="btn btn-secondary" onClick={() => lib.act.setVis(ids, 'public')}>
                <Icon name="globe" />
                设为公开
              </button>
              <button type="button" className="btn btn-secondary" onClick={() => lib.act.setVis(ids, 'private')}>
                <Icon name="lock" />
                设为私有
              </button>
            </div>
          </section>
        )}
      </div>
      <div className="detail-foot">
        {trash ? (
          <>
            <button type="button" className="btn btn-secondary" onClick={() => lib.act.restore(ids)}>
              <Icon name="restore" />
              恢复 {imgs.length} 张
            </button>
            <span className="grow" />
            <button type="button" className="btn btn-danger" onClick={() => lib.act.purge(ids)}>
              永久删除
            </button>
          </>
        ) : (
          <>
            <span className="grow" />
            <button type="button" className="btn btn-danger" onClick={() => lib.act.trash(ids)}>
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

function Dropzone({ lib, row }) {
  const s = lib.settings;
  return (
    <button type="button" className={`dropzone ${row ? 'row-mode' : ''}`} onClick={lib.pick}>
      <Icon name="upload" size={row ? 22 : 26} />
      <span className="dz-text">
        <span className="dz-main">
          拖放或粘贴图片，或 <span className="dz-link">选择文件</span>
        </span>
        <span className="hint">
          {s.allowedFormats.map((f) => FORMAT_LABEL[f]).join(' · ')}，单个不超过 {s.maxUploadMB} MB
        </span>
      </span>
    </button>
  );
}

function UploadVisibility({ lib }) {
  return <Seg label="上传后可见性" value={lib.uploadVis} onChange={lib.setUploadVis} options={VIS_OPTS} />;
}

function UploadList({ lib, items }) {
  const list = items ?? lib.uploads;
  return (
    <ul className="up-list">
      {list.map((u) => {
        const img = u.imageId ? lib.byId[u.imageId] : null;
        const thumbSrc = { thumb: u.thumb, format: u.format };
        return (
          <li className="up-row" key={u.id}>
            <Thumb image={img ?? thumbSrc} cover />
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
              {u.status === 'done' && <span className="sub">已上传 · {fmtBytes(u.size)}{img?.visibility === 'private' ? ' · 私有' : ''}</span>}
              {u.status === 'duplicate' && <span className="sub">已存在相同图片，返回原链接</span>}
              {u.status === 'error' && <span className="sub err">{u.error}</span>}
            </div>
            <div className="up-act">
              {(u.status === 'uploading' || u.status === 'queued' || u.status === 'processing') && (
                <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label={`取消上传 ${u.name}`} onClick={() => lib.cancelUpload(u.id)}>
                  <Icon name="x" />
                </button>
              )}
              {(u.status === 'done' || u.status === 'duplicate') && img && <CopyButton iconOnly size="sm" text={img.links[lib.linkFormat]} label={`复制 ${fmtLabel(lib.linkFormat)}`} />}
              {u.status === 'error' &&
                (u.retryable ? (
                  <button type="button" className="btn btn-secondary btn-sm" onClick={() => lib.retryUpload(u.id)}>
                    重试
                  </button>
                ) : (
                  <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label={`移除 ${u.name}`} onClick={() => lib.removeUpload(u.id)}>
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

/** Direction A: upload dialog. */
function UploadDialog({ lib, onClose }) {
  const s = uploadSummary(lib.uploads);
  return (
    <Modal
      title="上传图片"
      onClose={onClose}
      width="560px"
      className="full-sm"
      footer={
        <>
          {s.done > 0 && (
            <button type="button" className="btn btn-ghost" onClick={lib.clearFinished} style={{ marginRight: 'auto' }}>
              清除已完成
            </button>
          )}
          {s.done > 1 && (
            <button type="button" className="btn btn-secondary" onClick={() => copyFinished(lib)}>
              <Icon name="copy" />
              复制全部链接
            </button>
          )}
          <button type="button" className="btn btn-primary" onClick={onClose}>
            {s.active ? '在后台继续' : '完成'}
          </button>
        </>
      }
    >
      <Dropzone lib={lib} />
      <div className="setting-row" style={{ marginTop: 12 }}>
        <span className="label">上传后可见性</span>
        <UploadVisibility lib={lib} />
      </div>
      {lib.uploads.length > 0 && (
        <>
          <div className="section-label" style={{ marginTop: 16, marginBottom: 0 }}>
            {s.active ? `正在上传 ${s.active} 个，已完成 ${s.done} 个` : `已完成 ${s.done} 个`}
            {s.failed ? `，失败 ${s.failed} 个` : ''}
          </div>
          <UploadList lib={lib} />
        </>
      )}
    </Modal>
  );
}

/** Direction B: collapsible tray in the corner. */
function UploadTray({ lib, onClose }) {
  const [open, setOpen] = React.useState(true);
  const s = uploadSummary(lib.uploads);
  if (!lib.uploads.length) return null;
  return (
    <div className="tray" role="region" aria-label="上传">
      <div className="tray-head">
        <span>{s.active ? `正在上传 ${s.done}/${s.total}` : `已上传 ${s.done} 个`}</span>
        {s.failed > 0 && <span className="badge badge-danger">失败 {s.failed}</span>}
        <span className="grow" />
        <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label={open ? '收起' : '展开'} aria-expanded={open} onClick={() => setOpen(!open)}>
          <Icon name={open ? 'chevron-down' : 'chevron-up'} />
        </button>
        {!s.active && (
          <button type="button" className="btn btn-ghost btn-icon btn-sm" aria-label="关闭" onClick={onClose}>
            <Icon name="x" />
          </button>
        )}
      </div>
      {open && (
        <>
          <div className="tray-body">
            <UploadList lib={lib} />
          </div>
          {s.done > 0 && (
            <div className="tray-foot">
              <button type="button" className="btn btn-secondary btn-sm" onClick={() => copyFinished(lib)}>
                <Icon name="copy" />
                复制全部链接
              </button>
              <span className="grow" />
              <span className="hint">可见性</span>
              <UploadVisibility lib={lib} />
            </div>
          )}
        </>
      )}
    </div>
  );
}

/** Direction C: upload + recent uploads panel at the top of the library. */
function QuickUpload({ lib }) {
  const recent = [...lib.images.filter((i) => i.state === 'active')].sort((a, b) => b.createdAt - a.createdAt).slice(0, 5);
  const pending = lib.uploads.filter((u) => u.status !== 'done' && u.status !== 'duplicate');
  return (
    <div className="quick">
      <div className="quick-up">
        <Dropzone lib={lib} />
        <div className="quick-opts">
          <span>可见性</span>
          <UploadVisibility lib={lib} />
          <span className="quick-gap" />
          <span>复制格式</span>
          <Menu
            trigger={(open, toggle) => (
              <button type="button" className="btn btn-secondary btn-sm" aria-expanded={open} onClick={toggle}>
                {fmtLabel(lib.linkFormat)}
                <Icon name="chevron-down" size={14} />
              </button>
            )}
          >
            {(close) =>
              LINK_FORMATS.map((f) => (
                <RadioItem key={f.key} on={lib.linkFormat === f.key} onClick={() => (lib.setLinkFormat(f.key), close())}>
                  {f.label}
                </RadioItem>
              ))
            }
          </Menu>
        </div>
      </div>
      <div className="card recent">
        <div className="card-head">
          <h2>最近上传</h2>
          {lib.uploads.length > 0 && (
            <button type="button" className="btn btn-ghost btn-sm" onClick={lib.clearFinished}>
              清除记录
            </button>
          )}
        </div>
        {pending.length > 0 && (
          <div className="recent-pending">
            <UploadList lib={lib} items={pending} />
          </div>
        )}
        <div className="recent-list">
          {lib.dataState === 'loading'
            ? Array.from({ length: 4 }, (_, i) => (
                <div className="recent-row" key={i}>
                  <div className="skeleton" style={{ width: 36, height: 36 }} />
                  <div className="skeleton" style={{ height: 12, width: '50%' }} />
                </div>
              ))
            : recent.map((im) => (
                <div className="recent-row" key={im.id}>
                  <Thumb image={im} cover />
                  <button type="button" className="recent-main" onClick={() => lib.open(im.id)}>
                    <span className="name truncate">{im.filename}</span>
                    <span className="sub truncate mono">{im.links[lib.linkFormat]}</span>
                  </button>
                  <CopyButton size="sm" text={im.links[lib.linkFormat]} label="复制" />
                </div>
              ))}
          {lib.dataState !== 'loading' && !recent.length && !pending.length && <p className="recent-empty">上传的图片会显示在这里</p>}
        </div>
      </div>
    </div>
  );
}

function PurgeConfirm({ count, onCancel, onConfirm }) {
  return (
    <Modal
      title={count === 1 ? '永久删除这张图片？' : `永久删除 ${count} 张图片？`}
      onClose={onCancel}
      width="420px"
      footer={
        <>
          <button type="button" className="btn btn-secondary" onClick={onCancel} autoFocus>
            取消
          </button>
          <button type="button" className="btn btn-danger-solid" onClick={onConfirm}>
            永久删除
          </button>
        </>
      }
    >
      <p>文件会立即从磁盘删除，已发出的链接将失效。此操作无法撤销。</p>
    </Modal>
  );
}

function DropOverlay() {
  return (
    <div className="drop-overlay" aria-hidden="true">
      <div>
        <Icon name="upload" size={28} />
        <strong>松开以上传</strong>
        <span className="hint">可以一次放入多张图片</span>
      </div>
    </div>
  );
}

Object.assign(window, {
  FORMATS, VIS_OPTS, toggleIn, countWhere, fmtLabel, RadioItem, FilterMenus, FilterChips, FacetPanel, FilterSheet, SortMenu,
  ViewToggle, SearchBox, LibraryToolbar, BulkBar, RowMenu, ImageTable, ImageGrid, ImageRows, ImageCollection,
  DetailPanel, BatchPanel, Dropzone, UploadList, UploadDialog, UploadTray, QuickUpload, PurgeConfirm, DropOverlay,
  uploadSummary, copyMany,
});
