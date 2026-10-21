// Icons and presentational primitives shared by all directions.

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
  menu: ['M4 7h16', 'M4 12h16', 'M4 17h16'],
  external: ['M14 4h6v6', 'm20 4-9 9', 'M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5'],
  tag: ['M4 4h7.5l8.5 8.5-7.5 7.5L4 11.5z', 'M8.5 8.5h.01'],
  pin: ['M12 21s6.5-6 6.5-11a6.5 6.5 0 0 0-13 0c0 5 6.5 11 6.5 11z', 'M12 7.5a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5z'],
  sort: ['M7 4v16', 'm4 7 3-3 3 3', 'M17 20V4', 'm14 17 3 3 3-3'],
  plus: ['M12 5v14', 'M5 12h14'],
  alert: ['M12 3.5 21 19.5H3z', 'M12 10v4', 'M12 17h.01'],
  info: ['M12 3.5a8.5 8.5 0 1 0 0 17 8.5 8.5 0 0 0 0-17z', 'M12 11v5', 'M12 8h.01'],
  sun: ['M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8z', 'M12 2.5v2M12 19.5v2M2.5 12h2M19.5 12h2M5.3 5.3l1.4 1.4M17.3 17.3l1.4 1.4M5.3 18.7l1.4-1.4M17.3 6.7l1.4-1.4'],
  moon: ['M19.5 14.5A8 8 0 0 1 9.5 4.5a8 8 0 1 0 10 10z'],
  logout: ['M14 4h5a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-5', 'M10 8l-4 4 4 4', 'M6 12h10'],
  edit: ['M4 20h4L19 9l-4-4L4 16z', 'm13.5 6.5 4 4'],
  clipboard: ['M9 4h6v3H9z', 'M15 5.5h3v14.5H6V5.5h3'],
  folder: ['M3.5 6.5h6l2 2h9v10h-17z'],
  refresh: ['M20 11a8 8 0 0 0-14.5-4.5', 'M4 4v4h4', 'M4 13a8 8 0 0 0 14.5 4.5', 'M20 20v-4h-4'],
};

function Icon({ name, size = 16 }) {
  const d = ICON_PATHS[name] || [];
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false" style={{ flex: 'none' }}>
      {d.map((p, i) => (
        <path key={i} d={p} />
      ))}
    </svg>
  );
}

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

function useOutside(ref, onOut, active) {
  React.useEffect(() => {
    if (!active) return;
    const down = (e) => {
      if (ref.current && !ref.current.contains(e.target)) onOut();
    };
    const key = (e) => e.key === 'Escape' && onOut();
    document.addEventListener('pointerdown', down);
    document.addEventListener('keydown', key);
    return () => {
      document.removeEventListener('pointerdown', down);
      document.removeEventListener('keydown', key);
    };
  }, [active]);
}

/** Dropdown: `trigger(open, toggle)` renders the button; children receive `close`. */
function Menu({ trigger, children, align = 'left', up = false, width }) {
  const [open, setOpen] = React.useState(false);
  const ref = React.useRef(null);
  useOutside(ref, () => setOpen(false), open);
  return (
    <div className="menu-wrap" ref={ref}>
      {trigger(open, () => setOpen((o) => !o))}
      {open && (
        <div className={`menu ${align === 'right' ? 'right' : ''} ${up ? 'up' : ''}`} role="menu" style={width ? { minWidth: width } : undefined}>
          {typeof children === 'function' ? children(() => setOpen(false)) : children}
        </div>
      )}
    </div>
  );
}

function MenuItem({ icon, children, onClick, danger, end, checked }) {
  return (
    <button type="button" role={checked === undefined ? 'menuitem' : 'menuitemcheckbox'} aria-checked={checked} className={`menu-item ${danger ? 'danger' : ''}`} onClick={onClick}>
      {checked !== undefined ? <input type="checkbox" className="check" checked={checked} readOnly tabIndex={-1} /> : icon ? <Icon name={icon} /> : null}
      <span className="truncate">{children}</span>
      {end !== undefined && <span className="end">{end}</span>}
    </button>
  );
}

function Switch({ checked, onChange, label, disabled }) {
  return <button type="button" role="switch" className="switch" aria-checked={checked} aria-label={label} disabled={disabled} onClick={() => onChange(!checked)} />;
}

function Seg({ options, value, onChange, label, icons, className = '' }) {
  return (
    <div className={`seg ${icons ? 'seg-icons' : ''} ${className}`} role="group" aria-label={label}>
      {options.map((o) => (
        <button key={o.key} type="button" aria-pressed={value === o.key} onClick={() => onChange(o.key)} title={o.title} aria-label={icons ? o.title : undefined}>
          {icons ? <Icon name={o.icon} /> : o.label}
        </button>
      ))}
    </div>
  );
}

function useEsc(onEsc, active = true) {
  React.useEffect(() => {
    if (!active) return;
    const k = (e) => e.key === 'Escape' && onEsc();
    document.addEventListener('keydown', k);
    return () => document.removeEventListener('keydown', k);
  }, [active, onEsc]);
}

function Modal({ title, onClose, children, footer, width, className = '' }) {
  useEsc(onClose);
  return (
    <>
      <div className="scrim" onClick={onClose} />
      <div className={`modal ${className}`} role="dialog" aria-modal="true" aria-label={title} style={width ? { '--w': width } : undefined}>
        {title !== null && (
          <div className="modal-head">
            <h2>{title}</h2>
            <button type="button" className="btn btn-ghost btn-icon btn-sm" onClick={onClose} aria-label="关闭">
              <Icon name="x" />
            </button>
          </div>
        )}
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-foot">{footer}</div>}
      </div>
    </>
  );
}

function Drawer({ onClose, children, width, side = 'right', label }) {
  useEsc(onClose);
  return (
    <>
      <div className="scrim" onClick={onClose} />
      <aside className={`drawer ${side === 'left' ? 'left' : ''}`} role="dialog" aria-modal="true" aria-label={label} style={width ? { '--w': width } : undefined}>
        {children}
      </aside>
    </>
  );
}

function Sheet({ onClose, children, label }) {
  useEsc(onClose);
  return (
    <>
      <div className="scrim" onClick={onClose} />
      <div className="sheet" role="dialog" aria-modal="true" aria-label={label}>
        <div className="sheet-grip" />
        {children}
      </div>
    </>
  );
}

function Thumb({ image, className = '', cover }) {
  const alpha = image.format === 'png' || image.format === 'webp' || image.format === 'gif';
  return (
    <span className={`thumb ${alpha ? 'checker' : ''} ${cover ? 'cover' : ''} ${className}`}>
      <img src={image.thumb} alt="" loading="lazy" draggable="false" />
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
    <button type="button" className={`btn ${className} ${size === 'sm' ? 'btn-sm' : ''} ${iconOnly ? 'btn-icon' : ''}`} onClick={run} aria-label={iconOnly ? (done ? '已复制' : label) : undefined} title={iconOnly ? (done ? '已复制' : label) : undefined}>
      <Icon name={done ? 'check' : 'copy'} />
      {!iconOnly && <span>{done ? '已复制' : label}</span>}
    </button>
  );
}

/** Split button: copy in the current format, menu to pick another. */
function CopyLinkSplit({ image, format, onFormat, size = 'sm' }) {
  const label = LINK_FORMATS.find((f) => f.key === format)?.label;
  return (
    <div className="btn-group" onClick={(e) => e.stopPropagation()}>
      <CopyButton text={image.links[format]} label={`复制 ${label}`} size={size} />
      <Menu
        align="right"
        trigger={(open, toggle) => (
          <button type="button" className={`btn btn-secondary btn-icon ${size === 'sm' ? 'btn-sm' : ''}`} aria-label="选择链接格式" aria-expanded={open} onClick={toggle}>
            <Icon name="chevron-down" size={14} />
          </button>
        )}
      >
        {(close) => (
          <>
            <div className="menu-label">复制为</div>
            {LINK_FORMATS.map((f) => (
              <MenuItem
                key={f.key}
                icon={f.key === format ? 'check' : undefined}
                onClick={() => {
                  writeClipboard(image.links[f.key]);
                  onFormat(f.key);
                  close();
                }}
              >
                {f.label}
              </MenuItem>
            ))}
          </>
        )}
      </Menu>
    </div>
  );
}

function VisBadge({ v }) {
  return v === 'public' ? (
    <span className="badge badge-success">
      <span className="dot" />
      公开
    </span>
  ) : (
    <span className="badge">
      <Icon name="lock" size={12} />
      私有
    </span>
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

function Brand() {
  return (
    <a className="brand" href="#" onClick={(e) => e.preventDefault()}>
      <span className="brand-mark">
        <Icon name="image" size={14} />
      </span>
      <span>PicBerth</span>
    </a>
  );
}

Object.assign(window, { Icon, useMedia, useOutside, useEsc, Menu, MenuItem, Switch, Seg, Modal, Drawer, Sheet, Thumb, CopyButton, CopyLinkSplit, VisBadge, Toasts, Brand, writeClipboard });
