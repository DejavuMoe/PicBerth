// Shared primitives: icons, floating layers, controls, toasts, dialogs.

const { useState, useEffect, useRef, useMemo, useCallback, useLayoutEffect, Fragment } = React;

// ---------------------------------------------------------------- icons (16px grid, 1.5 stroke)
const PATHS = {
  image: <><rect x="2.5" y="2.5" width="11" height="11" rx="2" /><circle cx="6" cy="6.25" r="1.1" /><path d="M13.5 10.5 10.5 7.5 3.5 13.5" /></>,
  trash: <><path d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.5 8.5h6l.5-8.5" /></>,
  key: <><circle cx="5.5" cy="10.5" r="2.5" /><path d="M7.3 8.7 13 3M11 5l1.5 1.5M9.5 6.5l1.2 1.2" /></>,
  sliders: <><path d="M3 5h6M12 5h1M3 11h1M7 11h6" /><circle cx="10.5" cy="5" r="1.5" /><circle cx="5.5" cy="11" r="1.5" /></>,
  pulse: <><path d="M2 8h2.5L6 4.5l3 7L10.5 8H14" /></>,
  search: <><circle cx="7" cy="7" r="4.25" /><path d="m10.25 10.25 3.25 3.25" /></>,
  filter: <><path d="M2.5 4h11M4.5 8h7M6.5 12h3" /></>,
  upload: <><path d="M8 10.5V2.75M4.75 6 8 2.75 11.25 6M3 10v2.5c0 .55.45 1 1 1h8c.55 0 1-.45 1-1V10" /></>,
  copy: <><rect x="5.5" y="5.5" width="8" height="8" rx="1.5" /><path d="M10.5 5.5V4a1.5 1.5 0 0 0-1.5-1.5H4A1.5 1.5 0 0 0 2.5 4v5A1.5 1.5 0 0 0 4 10.5h1.5" /></>,
  check: <><path d="m3.5 8.5 3 3 6-7" /></>,
  minus: <><path d="M4 8h8" /></>,
  x: <><path d="m4 4 8 8M12 4l-8 8" /></>,
  more: <><circle cx="3.5" cy="8" r=".6" /><circle cx="8" cy="8" r=".6" /><circle cx="12.5" cy="8" r=".6" /></>,
  lock: <><rect x="3.5" y="7" width="9" height="6.5" rx="1.5" /><path d="M5.5 7V5.25a2.5 2.5 0 0 1 5 0V7" /></>,
  globe: <><circle cx="8" cy="8" r="5.5" /><path d="M2.5 8h11M8 2.5c1.5 1.6 2.25 3.4 2.25 5.5S9.5 11.9 8 13.5C6.5 11.9 5.75 10.1 5.75 8S6.5 4.1 8 2.5Z" /></>,
  left: <><path d="M10 3.5 5.5 8l4.5 4.5" /></>,
  right: <><path d="M6 3.5 10.5 8 6 12.5" /></>,
  down: <><path d="M4 6l4 4 4-4" /></>,
  list: <><path d="M5.5 4h8M5.5 8h8M5.5 12h8M2.5 4h.01M2.5 8h.01M2.5 12h.01" /></>,
  grid: <><rect x="2.5" y="2.5" width="4.5" height="4.5" rx="1" /><rect x="9" y="2.5" width="4.5" height="4.5" rx="1" /><rect x="2.5" y="9" width="4.5" height="4.5" rx="1" /><rect x="9" y="9" width="4.5" height="4.5" rx="1" /></>,
  sort: <><path d="M5 3v10M2.5 10.5 5 13l2.5-2.5M11 13V3M8.5 5.5 11 3l2.5 2.5" /></>,
  download: <><path d="M8 2.75v7.75M4.75 7.25 8 10.5l3.25-3.25M3 13.25h10" /></>,
  restore: <><path d="M3 8a5 5 0 1 0 1.5-3.55M3 2.75V5h2.25" /></>,
  plus: <><path d="M8 3v10M3 8h10" /></>,
  select: <><rect x="2.5" y="2.5" width="11" height="11" rx="2" /><path d="m5.5 8 1.75 1.75L10.75 6" /></>,
  sun: <><circle cx="8" cy="8" r="2.75" /><path d="M8 1.75v1.5M8 12.75v1.5M1.75 8h1.5M12.75 8h1.5M3.6 3.6l1 1M11.4 11.4l1 1M3.6 12.4l1-1M11.4 4.6l1-1" /></>,
  logout: <><path d="M9.5 3H4a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h5.5M7 8h6.5M11 5.5 13.5 8 11 10.5" /></>,
  alert: <><circle cx="8" cy="8" r="5.5" /><path d="M8 5v3.25M8 10.75h.01" /></>,
  refresh: <><path d="M13 8a5 5 0 1 1-1.5-3.55M13 2.75V5h-2.25" /></>,
  edit: <><path d="M10.5 3 13 5.5 6 12.5H3.5V10Z" /></>,
  folder: <><path d="M2.5 4.5a1 1 0 0 1 1-1h2.9l1.2 1.5h4.9a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1Z" /></>,
  'folder-plus': <><path d="M8.5 12.5h-5a1 1 0 0 1-1-1v-7a1 1 0 0 1 1-1h2.9l1.2 1.5h4.9a1 1 0 0 1 1 1V8M12 10v4M10 12h4" /></>,
  command: <><path d="M5.5 5.5h5v5h-5zM5.5 5.5V4a1.5 1.5 0 1 0-1.5 1.5h1.5ZM10.5 5.5H12A1.5 1.5 0 1 0 10.5 4v1.5ZM10.5 10.5V12a1.5 1.5 0 1 0 1.5-1.5h-1.5ZM5.5 10.5H4A1.5 1.5 0 1 0 5.5 12v-1.5Z" /></>,
  expand: <><path d="M9.5 2.5h4v4M13.5 2.5 9 7M6.5 13.5h-4v-4M2.5 13.5 7 9" /></>,
  up: <><path d="M4 10l4-4 4 4" /></>,
  enter: <><path d="M13 3.5v4a2 2 0 0 1-2 2H3.5M6 7 3.5 9.5 6 12" /></>,
  shield: <><path d="M8 2 3 4v3.5c0 3 2 5.3 5 6.5 3-1.2 5-3.5 5-6.5V4Z" /><path d="m6 8 1.5 1.5L10 6.5" /></>,
  scan: <><path d="M2.5 5.5v-2a1 1 0 0 1 1-1h2M10.5 2.5h2a1 1 0 0 1 1 1v2M13.5 10.5v2a1 1 0 0 1-1 1h-2M5.5 13.5h-2a1 1 0 0 1-1-1v-2M5 8h6" /></>,
  archive: <><rect x="2.5" y="3" width="11" height="3.5" rx="1" /><path d="M3.5 6.5V12a1 1 0 0 0 1 1h7a1 1 0 0 0 1-1V6.5M6.5 9.5h3" /></>,
  user: <><circle cx="8" cy="5.75" r="2.75" /><path d="M3 13.5c.6-2.3 2.5-3.5 5-3.5s4.4 1.2 5 3.5" /></>,
  moon: <><path d="M13 9.5A5.5 5.5 0 0 1 6.5 3 5.5 5.5 0 1 0 13 9.5Z" /></>,
  monitor: <><rect x="2" y="3" width="12" height="8" rx="1.5" /><path d="M6 13.5h4M8 11v2.5" /></>,
  book: <><path d="M3 3.5A1.5 1.5 0 0 1 4.5 2H13v9.5H4.5A1.5 1.5 0 0 0 3 13V3.5ZM3 13a1.5 1.5 0 0 0 1.5 1.5H13" /></>,
  external: <><path d="M9 3h4v4M13 3 7.5 8.5M11 9.5V12a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h2.5" /></>,
};
function Icon({ name, size }) {
  return (
    <svg className={'icon' + (size === 's' ? ' icon-s' : '')} viewBox="0 0 16 16" aria-hidden="true">
      {PATHS[name]}
    </svg>
  );
}

// ---------------------------------------------------------------- media
function useMedia(query) {
  const [m, setM] = useState(() => matchMedia(query).matches);
  useEffect(() => {
    const mq = matchMedia(query);
    const on = () => setM(mq.matches);
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  }, [query]);
  return m;
}

// ---------------------------------------------------------------- floating
function useFloating(open, anchorRef, floatRef, align = 'end') {
  const [pos, setPos] = useState(null);
  useLayoutEffect(() => {
    if (!open) return setPos(null);
    const place = () => {
      const a = anchorRef.current?.getBoundingClientRect();
      const f = floatRef.current?.getBoundingClientRect();
      if (!a || !f) return;
      const vw = innerWidth;
      const vh = innerHeight;
      let left = align === 'end' ? a.right - f.width : a.left;
      left = Math.max(8, Math.min(left, vw - f.width - 8));
      let top = a.bottom + 4;
      if (top + f.height > vh - 8) top = Math.max(8, a.top - f.height - 4);
      setPos({ left, top });
    };
    place();
    addEventListener('resize', place);
    addEventListener('scroll', place, true);
    return () => {
      removeEventListener('resize', place);
      removeEventListener('scroll', place, true);
    };
  }, [open, align]);
  return pos;
}

function useDismiss(open, refs, onClose) {
  useEffect(() => {
    if (!open) return;
    const down = (e) => {
      if (refs.some((r) => r.current && r.current.contains(e.target))) return;
      if (e.target.closest?.('[data-nested]')) return;
      onClose();
    };
    const key = (e) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onClose();
        refs[0].current?.focus?.();
      }
    };
    document.addEventListener('pointerdown', down, true);
    document.addEventListener('keydown', key, true);
    return () => {
      document.removeEventListener('pointerdown', down, true);
      document.removeEventListener('keydown', key, true);
    };
  }, [open]);
}

/** A trigger plus a floating layer. `trigger(props)` renders the button; `children(close)` the content. */
function Popover({ trigger, children, align = 'end', className = 'menu', label, nested }) {
  const [open, setOpen] = useState(false);
  const aRef = useRef(null);
  const fRef = useRef(null);
  const pos = useFloating(open, aRef, fRef, align);
  const close = () => setOpen(false);
  useDismiss(open, [aRef, fRef], close);
  useEffect(() => {
    if (open && pos) fRef.current?.querySelector('button:not(:disabled), input')?.focus({ preventScroll: true });
  }, [open, !!pos]);
  const onKey = (e) => {
    if (!['ArrowDown', 'ArrowUp'].includes(e.key)) return;
    const items = [...fRef.current.querySelectorAll('.menu-item:not(:disabled)')];
    if (!items.length) return;
    e.preventDefault();
    const i = items.indexOf(document.activeElement);
    const n = e.key === 'ArrowDown' ? (i + 1) % items.length : (i - 1 + items.length) % items.length;
    items[n].focus();
  };
  return (
    <>
      {trigger({ ref: aRef, onClick: () => setOpen((o) => !o), 'aria-expanded': open, 'aria-haspopup': 'true' })}
      {open &&
        ReactDOM.createPortal(
          <div
            ref={fRef}
            className={className}
            role={className === 'menu' ? 'menu' : 'dialog'}
            aria-label={label}
            data-nested={nested ? 'true' : undefined}
            onKeyDown={onKey}
            style={pos ? { left: pos.left, top: pos.top } : { left: -9999, top: 0 }}
          >
            {children(close)}
          </div>,
          document.body,
        )}
    </>
  );
}

function MenuItem({ icon, children, onClick, danger, end, checked, disabled, indent = 0 }) {
  return (
    <button
      type="button"
      className={'menu-item' + (danger ? ' danger' : '')}
      style={indent ? { paddingLeft: 8 + (indent - 1) * 16 } : undefined}
      role={checked === undefined ? 'menuitem' : 'menuitemradio'}
      aria-checked={checked}
      onClick={onClick}
      disabled={disabled}
    >
      {checked !== undefined && <span className="tick">{checked && <Icon name="check" />}</span>}
      {icon && <Icon name={icon} />}
      <span className="ellipsis">{children}</span>
      {end && <span className="end">{end}</span>}
    </button>
  );
}

/** Product-drawn select: a secondary button that opens a menu. */
function Select({ value, options, onChange, label, width }) {
  const cur = options.find((o) => o.value === value);
  return (
    <Popover
      align="start"
      label={label}
      trigger={(p) => (
        <button {...p} type="button" className="btn btn-secondary" style={{ width, justifyContent: 'space-between', fontWeight: 400 }} aria-label={label}>
          <span className="ellipsis">{cur?.label}</span>
          <Icon name="down" size="s" />
        </button>
      )}
    >
      {(close) =>
        options.map((o) => (
          <MenuItem key={o.value} checked={o.value === value} onClick={() => (onChange(o.value), close())}>
            {o.label}
          </MenuItem>
        ))
      }
    </Popover>
  );
}

// ---------------------------------------------------------------- controls
function Check({ on, onChange, label, className = '' }) {
  return (
    <button
      type="button"
      className={'check-hit ' + className}
      role="checkbox"
      aria-checked={on === 'mixed' ? 'mixed' : !!on}
      aria-label={label}
      onClick={(e) => {
        e.stopPropagation();
        onChange(e);
      }}
    >
      <span className="check" data-on={String(on)}>
        {on === 'mixed' ? <Icon name="minus" /> : on ? <Icon name="check" /> : null}
      </span>
    </button>
  );
}

function Seg({ value, options, onChange, label, block }) {
  return (
    <div className={'seg' + (block ? ' seg-block' : '')} role="group" aria-label={label}>
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          aria-pressed={o.value === value}
          aria-label={o.aria}
          title={o.aria}
          onClick={() => onChange(o.value)}
        >
          {o.icon && <Icon name={o.icon} size="s" />}
          {o.label}
        </button>
      ))}
    </div>
  );
}

function Switch({ on, onChange, label }) {
  return <button type="button" role="switch" className="switch" aria-checked={on} aria-label={label} onClick={() => onChange(!on)} />;
}

// ---------------------------------------------------------------- dialog
function Dialog({ title, children, footer, onClose, wide, dismissible = true }) {
  const ref = useRef(null);
  useEffect(() => {
    const prev = document.activeElement;
    (ref.current.querySelector('[data-autofocus]') || ref.current.querySelector('input, button'))?.focus();
    const key = (e) => {
      if (e.key === 'Escape' && dismissible) {
        e.stopPropagation();
        onClose();
      }
      if (e.key === 'Tab') {
        const els = [...ref.current.querySelectorAll('button:not(:disabled), input, [tabindex="0"]')];
        const first = els[0];
        const last = els[els.length - 1];
        if (e.shiftKey && document.activeElement === first) (e.preventDefault(), last.focus());
        else if (!e.shiftKey && document.activeElement === last) (e.preventDefault(), first.focus());
      }
    };
    document.addEventListener('keydown', key, true);
    return () => {
      document.removeEventListener('keydown', key, true);
      prev?.focus?.();
    };
  }, []);
  return ReactDOM.createPortal(
    <div className="scrim" onPointerDown={(e) => dismissible && e.target === e.currentTarget && onClose()}>
      <div ref={ref} className={'dialog' + (wide ? ' dialog-wide' : '')} role="dialog" aria-modal="true" aria-labelledby="dlg-title">
        <h2 className="dialog-head" id="dlg-title">
          {title}
        </h2>
        <div className="dialog-body">{children}</div>
        {footer && <div className="dialog-foot">{footer}</div>}
      </div>
    </div>,
    document.body,
  );
}

// ---------------------------------------------------------------- toasts
const ToastCtx = React.createContext(() => {});
function ToastHost({ children }) {
  const [items, setItems] = useState([]);
  const push = useCallback((text, action) => {
    const id = Math.random();
    setItems((xs) => [...xs.slice(-2), { id, text, action }]);
    setTimeout(() => setItems((xs) => xs.filter((x) => x.id !== id)), action ? 6000 : 2600);
  }, []);
  return (
    <ToastCtx.Provider value={push}>
      {children}
      <div className="toasts" role="status" aria-live="polite">
        {items.map((t) => (
          <div key={t.id} className="toast">
            <span>{t.text}</span>
            {t.action && (
              <button
                type="button"
                onClick={() => {
                  t.action.run();
                  setItems((xs) => xs.filter((x) => x.id !== t.id));
                }}
              >
                {t.action.label}
              </button>
            )}
          </div>
        ))}
      </div>
    </ToastCtx.Provider>
  );
}
const useToast = () => React.useContext(ToastCtx);

function copyText(text) {
  try {
    navigator.clipboard?.writeText(text);
  } catch (e) {}
}

// ---------------------------------------------------------------- theme
function applyTheme(pref) {
  const dark = pref === 'dark' || (pref === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.dataset.theme = dark ? 'dark' : 'light';
}
function useTheme() {
  const [pref, setPref] = useState(() => {
    const q = new URLSearchParams(location.search).get('theme');
    try {
      return q || localStorage.getItem('picberth.theme') || 'system';
    } catch (e) {
      return q || 'system';
    }
  });
  const first = useRef(true);
  useEffect(() => {
    if (first.current) applyTheme(pref);
    else applyTheme(pref);
    first.current = false;
    try {
      localStorage.setItem('picberth.theme', pref);
    } catch (e) {}
    const mq = matchMedia('(prefers-color-scheme: dark)');
    const on = () => applyTheme(pref);
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  }, [pref]);
  return [pref, setPref];
}
/** Run a state change inside a view transition where the browser has one and motion is allowed. */
function vt(fn) {
  const still = matchMedia('(prefers-reduced-motion: reduce)').matches;
  if (!document.startViewTransition || still) return fn();
  document.startViewTransition(() => ReactDOM.flushSync(fn));
}

/** A dashed frame that follows its box (stroke stays 1:1 at any size). */
function DashFrame({ radius = 16 }) {
  return (
    <svg className="frame" aria-hidden="true">
      <rect x="0" y="0" width="100%" height="100%" rx={radius} />
    </svg>
  );
}

const IS_MAC = /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
const MOD = IS_MAC ? '⌘' : 'Ctrl';

const THEMES = [
  { value: 'system', label: '跟随系统', icon: 'monitor' },
  { value: 'light', label: '浅色', icon: 'sun' },
  { value: 'dark', label: '深色', icon: 'moon' },
];

if (new URLSearchParams(location.search).has('clean')) document.documentElement.classList.add('clean');

Object.assign(window, {
  Icon, useMedia, Popover, MenuItem, Select, Check, Seg, Switch, Dialog, ToastHost, useToast, copyText, useTheme, THEMES,
  useFloating, useDismiss, vt, DashFrame, MOD, IS_MAC,
});
