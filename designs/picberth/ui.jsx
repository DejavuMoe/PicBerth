// Icons, hooks, presentational primitives and the shared console chrome (sidebar, rail, tab bar).

const ICON_PATHS = {
  upload: ['M12 15V4', 'M7.5 8.5 12 4l4.5 4.5', 'M4 15v4a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-4'],
  download: ['M12 4v11', 'M7.5 10.5 12 15l4.5-4.5', 'M4 15v4a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-4'],
  search: ['M11 4.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13z', 'm16 16 4 4'],
  filter: ['M4 6h16', 'M7 12h10', 'M10 18h4'],
  grid: ['M4 4h7v7H4z', 'M13 4h7v7h-7z', 'M4 13h7v7H4z', 'M13 13h7v7h-7z'],
  list: ['M9 6h11', 'M9 12h11', 'M9 18h11', 'M4.5 6h.01', 'M4.5 12h.01', 'M4.5 18h.01'],
  copy: ['M9 9h10v11H9z', 'M5 15H4V4h11v1'],
  check: ['m5 12.5 4.5 4.5L19 7.5'],
  more: ['M5 12h.01', 'M12 12h.01', 'M19 12h.01'],
  trash: ['M4 7h16', 'M9.5 7V4.5h5V7', 'M6 7l1 13h10l1-13', 'M10 11v6', 'M14 11v6'],
  restore: ['M4 12a8 8 0 1 0 2.4-5.7', 'M4 4v4.5h4.5'],
  lock: ['M5.5 11h13v9h-13z', 'M8.5 11V8a3.5 3.5 0 0 1 7 0v3'],
  globe: ['M12 3.5a8.5 8.5 0 1 0 0 17 8.5 8.5 0 0 0 0-17z', 'M3.5 12h17', 'M12 3.5c2.5 2.6 2.5 14.4 0 17', 'M12 3.5c-2.5 2.6-2.5 14.4 0 17'],
  x: ['m6 6 12 12', 'M18 6 6 18'],
  'chevron-down': ['m6.5 9.5 5.5 5.5 5.5-5.5'],
  'chevron-up': ['m6.5 14.5 5.5-5.5 5.5 5.5'],
  'chevron-left': ['m14.5 6.5-5.5 5.5 5.5 5.5'],
  'chevron-right': ['m9.5 6.5 5.5 5.5-5.5 5.5'],
  image: ['M4 5h16v14H4z', 'm4 16 4.5-4.5 4 4 2.5-2.5L20 18', 'M15.5 9.5h.01'],
  images: ['M7 7h13v11H7z', 'M4 4h13', 'M4 4v11'],
  key: ['M8 11.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z', 'm10.5 12.5 8-8', 'm16 7 2.5 2.5'],
  gear: ['M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z', 'M12 2.5v3M12 18.5v3M2.5 12h3M18.5 12h3M5.3 5.3l2.1 2.1M16.6 16.6l2.1 2.1M5.3 18.7l2.1-2.1M16.6 7.4l2.1-2.1'],
  activity: ['M3 12h4l2.5-6 5 12 2.5-6h4'],
  external: ['M14 4h6v6', 'm20 4-9 9', 'M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5'],
  sort: ['M7 4v16', 'm4 7 3-3 3 3', 'M17 20V4', 'm14 17 3 3 3-3'],
  plus: ['M12 5v14', 'M5 12h14'],
  alert: ['M12 3.5 21 19.5H3z', 'M12 10v4', 'M12 17h.01'],
  sun: ['M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8z', 'M12 2.5v2M12 19.5v2M2.5 12h2M19.5 12h2M5.3 5.3l1.4 1.4M17.3 17.3l1.4 1.4M5.3 18.7l1.4-1.4M17.3 6.7l1.4-1.4'],
  moon: ['M19.5 14.5A8 8 0 0 1 9.5 4.5a8 8 0 1 0 10 10z'],
  logout: ['M14 4h5a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-5', 'M10 8l-4 4 4 4', 'M6 12h10'],
  edit: ['M4 20h4L19 9l-4-4L4 16z', 'm13.5 6.5 4 4'],
  refresh: ['M20 11a8 8 0 0 0-14.5-4.5', 'M4 4v4h4', 'M4 13a8 8 0 0 0 14.5 4.5', 'M20 20v-4h-4'],
  select: ['M4.5 4.5h15v15h-15z', 'm8.5 12 2.5 2.5 4.5-5'],
};

// Dot glyphs need a heavier stroke to carry the same visual weight as line icons.
const ICON_STROKE = { more: 3.2 };

function Icon({ name, size = 16 }) {
  const d = ICON_PATHS[name] || [];
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={ICON_STROKE[name] ?? 1.7} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false" style={{ flex: 'none' }}>
      {d.map((p, i) => (
        <path key={i} d={p} />
      ))}
    </svg>
  );
}

// ---------------------------------------------------------------- hooks

function useMedia(query) {
  const [m, setM] = React.useState(() => window.matchMedia(query).matches);
  React.useEffect(() => {
    const mq = window.matchMedia(query);
    const on = () => setM(mq.matches);
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  }, [query]);
  return m;
}

/** Content width of an element, for layouts that depend on the space a component gets rather than the viewport.
 *  Returns [callbackRef, width]. */
function useWidth() {
  const [w, setW] = React.useState(0);
  const ro = React.useRef(null);
  const ref = React.useCallback((el) => {
    ro.current?.disconnect();
    ro.current = null;
    if (!el) return;
    ro.current = new ResizeObserver(([e]) => setW(Math.round(e.contentRect.width)));
    ro.current.observe(el);
  }, []);
  return [ref, w];
}

function useOutside(ref, onOut, active) {
  React.useEffect(() => {
    if (!active) return;
    const down = (e) => {
      if (ref.current && !ref.current.contains(e.target)) onOut();
    };
    const key = (e) => {
      if (e.key !== 'Escape') return;
      e.stopImmediatePropagation();
      onOut();
    };
    document.addEventListener('pointerdown', down);
    document.addEventListener('keydown', key, true);
    return () => {
      document.removeEventListener('pointerdown', down);
      document.removeEventListener('keydown', key, true);
    };
  }, [active]);
}

function useEsc(onEsc, active = true) {
  React.useEffect(() => {
    if (!active) return;
    const k = (e) => e.key === 'Escape' && onEsc();
    document.addEventListener('keydown', k);
    return () => document.removeEventListener('keydown', k);
  }, [active, onEsc]);
}

const REDUCED_MOTION = window.matchMedia('(prefers-reduced-motion: reduce)');

/** Plays the exit animation of an overlay before it unmounts. */
function useLeave(onClose, ms = 180) {
  const [leaving, setLeaving] = React.useState(false);
  const done = React.useRef(false);
  const close = React.useCallback(() => {
    if (done.current) return;
    done.current = true;
    if (REDUCED_MOTION.matches) return onClose();
    setLeaving(true);
    setTimeout(onClose, ms);
  }, [onClose, ms]);
  return [leaving, close];
}

// ---------------------------------------------------------------- menus

/** Dropdown: `trigger(open, toggle)` renders the button; children receive `close`. */
function Menu({ trigger, children, align = 'left', up = false, width, className = '', wrapClass = '' }) {
  const [open, setOpen] = React.useState(false);
  const ref = React.useRef(null);
  useOutside(ref, () => setOpen(false), open);
  return (
    <div className={`menu-wrap ${wrapClass}`} ref={ref}>
      {trigger(open, () => setOpen((o) => !o))}
      {open && (
        <div className={`menu ${align === 'right' ? 'right' : ''} ${up ? 'up' : ''} ${className}`} role="menu" style={width ? { minWidth: width } : undefined}>
          {typeof children === 'function' ? children(() => setOpen(false)) : children}
        </div>
      )}
    </div>
  );
}

function MenuItem({ icon, children, onClick, danger }) {
  return (
    <button type="button" role="menuitem" className={`menu-item ${danger ? 'danger' : ''}`} onClick={onClick}>
      {icon ? <Icon name={icon} /> : <span className="menu-tick" />}
      <span className="truncate">{children}</span>
    </button>
  );
}

function RadioItem({ on, children, onClick }) {
  return (
    <button type="button" role="menuitemradio" aria-checked={on} className="menu-item" onClick={onClick}>
      <span className="menu-tick">{on && <Icon name="check" />}</span>
      <span className="truncate">{children}</span>
    </button>
  );
}

// ---------------------------------------------------------------- controls

function Switch({ checked, onChange, label, disabled }) {
  return <button type="button" role="switch" className="switch" aria-checked={checked} aria-label={label} disabled={disabled} onClick={() => onChange(!checked)} />;
}

/** Segmented control. `icons`: icon-only buttons. An option with `icon` and a label shows both. */
function Seg({ options, value, onChange, label, icons, fill, className = '' }) {
  return (
    <div className={`seg ${icons ? 'seg-icons' : ''} ${fill ? 'fill' : ''} ${className}`} role="group" aria-label={label}>
      {options.map((o) => (
        <button key={o.key} type="button" aria-pressed={value === o.key} onClick={() => onChange(o.key)} title={o.title} aria-label={icons ? o.title : undefined}>
          {o.icon && <Icon name={o.icon} />}
          {!icons && o.label}
        </button>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------- overlays

const renderKids = (children, close) => (typeof children === 'function' ? children(close) : children);

function Modal({ title, onClose, children, footer, width, className = '' }) {
  const [leaving, close] = useLeave(onClose, 140);
  useEsc(close);
  return (
    <>
      <div className={`scrim ${leaving ? 'leaving' : ''}`} onClick={close} />
      <div className={`modal ${leaving ? 'leaving' : ''} ${className}`} role="dialog" aria-modal="true" aria-label={title || undefined} style={width ? { '--w': width } : undefined}>
        <div className="modal-head">
          <h2>{title}</h2>
          <button type="button" className="btn btn-ghost btn-icon btn-sm" onClick={close} aria-label="关闭">
            <Icon name="x" />
          </button>
        </div>
        <div className="modal-body">{renderKids(children, close)}</div>
        {footer && <div className="modal-foot">{renderKids(footer, close)}</div>}
      </div>
    </>
  );
}

function Drawer({ onClose, children, width, label }) {
  const [leaving, close] = useLeave(onClose, 200);
  useEsc(close);
  return (
    <>
      <div className={`scrim ${leaving ? 'leaving' : ''}`} onClick={close} />
      <aside className={`drawer ${leaving ? 'leaving' : ''}`} role="dialog" aria-modal="true" aria-label={label} style={width ? { '--w': width } : undefined}>
        {renderKids(children, close)}
      </aside>
    </>
  );
}

function Sheet({ onClose, children, label, full }) {
  const [leaving, close] = useLeave(onClose, 220);
  useEsc(close);
  return (
    <>
      <div className={`scrim ${leaving ? 'leaving' : ''}`} onClick={close} />
      <div className={`sheet ${full ? 'sheet-full' : ''} ${leaving ? 'leaving' : ''}`} role="dialog" aria-modal="true" aria-label={label}>
        <div className="sheet-grip" />
        {renderKids(children, close)}
      </div>
    </>
  );
}

// ---------------------------------------------------------------- content pieces

function hasAlpha(im) {
  return im.format === 'png' || im.format === 'webp' || im.format === 'gif';
}

function Thumb({ image, className = '' }) {
  return (
    <span className={`thumb ${className}`}>
      {image.thumb ? <img src={image.thumb} alt="" loading="lazy" draggable="false" /> : <Icon name="image" />}
    </span>
  );
}

async function writeClipboard(text) {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    // preview without clipboard permission: the toast still confirms the action
  }
}

function CopyButton({ text, label = '复制', size, className = 'btn-secondary', iconOnly, onCopied }) {
  const [done, setDone] = React.useState(false);
  const t = React.useRef(0);
  const run = async (e) => {
    e.stopPropagation();
    await writeClipboard(text);
    setDone(true);
    clearTimeout(t.current);
    t.current = setTimeout(() => setDone(false), 1400);
    onCopied?.();
  };
  return (
    <button
      type="button"
      className={`btn ${className} ${size === 'sm' ? 'btn-sm' : ''} ${iconOnly ? 'btn-icon' : ''} ${done ? 'copied' : ''}`}
      onClick={run}
      aria-label={iconOnly ? (done ? '已复制' : label) : undefined}
      title={iconOnly ? (done ? '已复制' : label) : undefined}
    >
      <Icon name={done ? 'check' : 'copy'} key={done ? 'done' : 'copy'} />
      {!iconOnly && <span>{done ? '已复制' : label}</span>}
    </button>
  );
}

function Toasts({ toasts, dismiss }) {
  return (
    <div className="toasts" role="status" aria-live="polite">
      {toasts.map((t) => (
        <div className="toast" key={t.id}>
          <span>{t.text}</span>
          {t.action && (
            <button
              type="button"
              onClick={() => {
                t.action.run();
                dismiss(t.id);
              }}
            >
              {t.action.label}
            </button>
          )}
        </div>
      ))}
    </div>
  );
}

function BrandMark({ size = 14, className = '' }) {
  return (
    <span className={`brand-mark ${className}`}>
      <Icon name="image" size={size} />
    </span>
  );
}

function Brand() {
  return (
    <a className="brand" href="#" onClick={(e) => e.preventDefault()} aria-label="PicBerth">
      <BrandMark />
      <span className="brand-name">PicBerth</span>
    </a>
  );
}

// ---------------------------------------------------------------- console chrome

const NAV = [
  { key: 'images', label: '全部图片', short: '图片', icon: 'images', group: 0 },
  { key: 'trash', label: '回收站', short: '回收站', icon: 'trash', group: 0 },
  { key: 'tokens', label: '令牌', short: '令牌', icon: 'key', group: 1 },
  { key: 'settings', label: '设置', short: '设置', icon: 'gear', group: 2 },
  { key: 'status', label: '状态', short: '状态', icon: 'activity', group: 2 },
];
const TAB_KEYS = ['images', 'trash', 'tokens'];

function ThemeItem({ theme, setTheme, close }) {
  const dark = theme === 'dark';
  return (
    <MenuItem icon={dark ? 'sun' : 'moon'} onClick={() => (setTheme(dark ? 'light' : 'dark'), close())}>
      {dark ? '切换到浅色' : '切换到深色'}
    </MenuItem>
  );
}

function AccountMenu({ theme, setTheme, toast }) {
  return (
    <Menu
      up
      width={200}
      wrapClass="account-wrap"
      trigger={(open, toggle) => (
        <button type="button" className="account" aria-label="账户" title="admin" aria-expanded={open} onClick={toggle}>
          <span className="avatar">A</span>
          <span className="account-name">admin</span>
        </button>
      )}
    >
      {(close) => (
        <>
          <ThemeItem theme={theme} setTheme={setTheme} close={close} />
          <MenuItem icon="logout" onClick={() => (close(), toast('已退出登录'))}>
            退出登录
          </MenuItem>
        </>
      )}
    </Menu>
  );
}

/** Desktop sidebar; becomes the icon rail through CSS on tablets. */
function Sidebar({ route, go, status, theme, setTheme, toast }) {
  const warn = status.warnings.length > 0;
  return (
    <aside className="sidenav" aria-label="主导航">
      <Brand />
      <nav>
        {NAV.map((n, i) => (
          <React.Fragment key={n.key}>
            {i > 0 && NAV[i - 1].group !== n.group && <div className="nav-gap" />}
            <button type="button" className="nav-item" aria-current={route === n.key ? 'page' : undefined} title={n.label} onClick={() => go(n.key)}>
              <Icon name={n.icon} />
              <span className="nav-label">{n.label}</span>
              {n.key === 'status' && warn && <span className="warn-dot" role="img" aria-label="有警告" />}
            </button>
          </React.Fragment>
        ))}
      </nav>
      <AccountMenu theme={theme} setTheme={setTheme} toast={toast} />
    </aside>
  );
}

function TabBar({ route, go, status, onMore, moreOpen }) {
  const moreOn = moreOpen || (route && !TAB_KEYS.includes(route));
  return (
    <nav className="tabbar" aria-label="主导航">
      {NAV.filter((n) => TAB_KEYS.includes(n.key)).map((n) => (
        <button key={n.key} type="button" className="tab" aria-current={route === n.key && !moreOpen ? 'page' : undefined} onClick={() => go(n.key)}>
          <Icon name={n.icon} size={20} />
          <span>{n.short}</span>
        </button>
      ))}
      <button type="button" className="tab" aria-current={moreOn ? 'page' : undefined} aria-expanded={moreOpen} onClick={onMore}>
        <span className="tab-icon">
          <Icon name="more" size={20} />
          {status.warnings.length > 0 && <span className="warn-dot" aria-hidden="true" />}
        </span>
        <span>更多</span>
      </button>
    </nav>
  );
}

function MoreSheet({ route, go, status, theme, setTheme, toast, onClose }) {
  return (
    <Sheet onClose={onClose} label="更多">
      {(close) => (
        <div className="sheet-body more-body">
          {NAV.filter((n) => !TAB_KEYS.includes(n.key)).map((n) => (
            <button
              key={n.key}
              type="button"
              className="nav-item"
              aria-current={route === n.key ? 'page' : undefined}
              onClick={() => {
                go(n.key);
                close();
              }}
            >
              <Icon name={n.icon} />
              <span className="nav-label">{n.label}</span>
              {n.key === 'status' && status.warnings.length > 0 && <span className="warn-dot" role="img" aria-label="有警告" />}
            </button>
          ))}
          <div className="setting-row more-row">
            <span className="label">深色模式</span>
            <Switch checked={theme === 'dark'} label="深色模式" onChange={(v) => setTheme(v ? 'dark' : 'light')} />
          </div>
          <button type="button" className="nav-item" onClick={() => (close(), toast('已退出登录'))}>
            <Icon name="logout" />
            <span className="nav-label">退出登录</span>
          </button>
        </div>
      )}
    </Sheet>
  );
}

/** Sticky page header: title on the left, page actions (children) on the right. */
function PageHead({ title, children }) {
  return (
    <header className="page-head">
      <h1>{title}</h1>
      {children && <div className="page-actions">{children}</div>}
    </header>
  );
}

Object.assign(window, {
  Icon, useMedia, useWidth, useOutside, useEsc, useLeave, Menu, MenuItem, RadioItem, Switch, Seg, Modal, Drawer, Sheet, Thumb,
  CopyButton, Toasts, Brand, BrandMark, writeClipboard, hasAlpha, NAV, TAB_KEYS, Sidebar, TabBar, MoreSheet, PageHead,
});
