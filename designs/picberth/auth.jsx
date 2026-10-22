// Setup, sign-in and not-found screens.

const AUTH_PARAMS = new URLSearchParams(location.search);
const CODE_CHARS = /[^0-9A-HJKMNP-TV-Z]/g;

function formatSetupCode(raw) {
  const chars = raw.toUpperCase().replace(CODE_CHARS, '').slice(0, 12);
  return chars.match(/.{1,4}/g)?.join('-') ?? '';
}

function AuthCard({ title, children }) {
  return (
    <main className="auth">
      <div className="auth-card">
        <Brand />
        <h1>{title}</h1>
        {children}
      </div>
    </main>
  );
}

function SetupScreen({ preset }) {
  const [code, setCode] = React.useState(preset === 'errors' ? 'K7Q2-M9' : preset === 'wrong' ? 'K7Q2-M9XD-4TRB' : '');
  const [username, setUsername] = React.useState('admin');
  const [password, setPassword] = React.useState(preset === 'errors' ? 'short' : preset === 'wrong' ? 'correct-horse-9' : '');
  const [confirm, setConfirm] = React.useState(preset === 'errors' ? 'shorter' : preset === 'wrong' ? 'correct-horse-9' : '');
  const [submitted, setSubmitted] = React.useState(preset === 'errors' || preset === 'wrong');
  const [busy, setBusy] = React.useState(preset === 'busy');
  const [formError, setFormError] = React.useState(preset === 'wrong' ? '初始化代码不正确' : preset === 'done' ? '管理员已经创建，请直接登录' : '');
  const full = code.replace(/-/g, '').length === 12;
  const errors = {
    code: !code ? '请输入服务器日志中的初始化代码' : full ? null : '初始化代码共 12 位，格式为 XXXX-XXXX-XXXX',
    username: /^[A-Za-z0-9._-]{1,64}$/.test(username.trim()) ? null : '1–64 个字符，可用字母、数字、点、下划线和连字符',
    password: password.length >= 10 && password.length <= 256 ? null : '密码需要 10–256 个字符',
    confirm: confirm === password ? null : '两次输入的密码不一致',
  };
  const valid = !errors.code && !errors.username && !errors.password && !errors.confirm;
  const submit = (e) => {
    e.preventDefault();
    setSubmitted(true);
    setFormError('');
    if (!valid) return;
    setBusy(true);
    setTimeout(() => {
      setBusy(false);
      setFormError(code === 'K7Q2-M9XD-4TRB' ? '初始化代码不正确' : '');
    }, 900);
  };
  const err = (k) => submitted && errors[k];
  return (
    <AuthCard title="初始化">
      <form className="auth-form" onSubmit={submit} noValidate>
        <div className="field">
          <label htmlFor="setup-code">初始化代码</label>
          <input id="setup-code" className="input code-input" value={code} onChange={(e) => setCode(formatSetupCode(e.target.value))} placeholder="XXXX-XXXX-XXXX" autoComplete="off" spellCheck="false" aria-invalid={err('code') ? 'true' : undefined} />
          <span className="hint">在服务器日志中找到初始化代码</span>
          {err('code') && <span className="error-text">{errors.code}</span>}
        </div>
        <div className="field">
          <label htmlFor="setup-user">用户名</label>
          <input id="setup-user" className="input" value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" spellCheck="false" aria-invalid={err('username') ? 'true' : undefined} />
          {err('username') && <span className="error-text">{errors.username}</span>}
        </div>
        <div className="field">
          <label htmlFor="setup-pass">密码</label>
          <input id="setup-pass" className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" aria-invalid={err('password') ? 'true' : undefined} />
          <span className="hint">至少 10 个字符</span>
          {err('password') && <span className="error-text">{errors.password}</span>}
        </div>
        <div className="field">
          <label htmlFor="setup-confirm">确认密码</label>
          <input id="setup-confirm" className="input" type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" aria-invalid={err('confirm') ? 'true' : undefined} />
          {err('confirm') && <span className="error-text">{errors.confirm}</span>}
        </div>
        {formError && (
          <div className="alert danger" role="alert">
            <Icon name="alert" />
            <span>{formError}</span>
          </div>
        )}
        <button type="submit" className="btn btn-primary btn-block" disabled={busy}>
          {busy ? '正在创建…' : '创建管理员'}
        </button>
      </form>
    </AuthCard>
  );
}

function LoginScreen({ preset }) {
  const [username, setUsername] = React.useState(preset === 'error' || preset === 'busy' ? 'admin' : '');
  const [password, setPassword] = React.useState(preset === 'busy' ? 'correct-horse-9' : '');
  const [busy, setBusy] = React.useState(preset === 'busy');
  const [error, setError] = React.useState(preset === 'error' ? '用户名或密码不正确' : preset === 'offline' ? '无法连接到服务器' : '');
  const submit = (e) => {
    e.preventDefault();
    if (busy) return;
    setBusy(true);
    setError('');
    setTimeout(() => {
      setBusy(false);
      setError('用户名或密码不正确');
      setPassword('');
    }, 900);
  };
  return (
    <AuthCard title="登录">
      <form className="auth-form" onSubmit={submit} noValidate>
        <div className="field">
          <label htmlFor="login-user">用户名</label>
          <input id="login-user" className="input" value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" spellCheck="false" />
        </div>
        <div className="field">
          <label htmlFor="login-pass">密码</label>
          <input id="login-pass" className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" />
        </div>
        {error && (
          <div className="alert danger" role="alert">
            <Icon name="alert" />
            <span>{error}</span>
          </div>
        )}
        <button type="submit" className="btn btn-primary btn-block" disabled={busy || !username || !password}>
          {busy ? '正在登录…' : '登录'}
        </button>
      </form>
    </AuthCard>
  );
}

function NotFoundScreen() {
  const mobile = useMedia('(max-width: 720px)');
  const rail = useMedia('(max-width: 1099px)') && !mobile;
  let group = null;
  return (
    <div className={`app ${rail ? 'rail' : ''} ${mobile ? 'mobile' : ''}`} data-screen-label="not-found">
      {!mobile && (
        <aside className="sidenav" aria-label="主导航">
          <Brand />
          <nav>
            {NAV.map((n) => {
              const head = n.group !== group ? <div className="nav-group">{(group = n.group)}</div> : null;
              return (
                <React.Fragment key={n.key}>
                  {head}
                  <button type="button" className="nav-item" title={n.label}>
                    <span className="nav-icon">
                      <Icon name={n.icon} />
                    </span>
                    <span className="nav-label">{n.label}</span>
                  </button>
                </React.Fragment>
              );
            })}
          </nav>
        </aside>
      )}
      <div className="main">
        <header className="topbar">
          {mobile ? (
            <Brand />
          ) : (
            <nav className="crumbs" aria-label="位置">
              <span>PicBerth</span>
              <Icon name="chevron-right" size={12} />
              <strong>页面不存在</strong>
            </nav>
          )}
        </header>
        <main>
          <div className="page" style={{ '--page-w': '720px' }}>
            <div className="empty">
              <span className="empty-icon">
                <Icon name="search" size={22} />
              </span>
              <strong>页面不存在</strong>
              <p>这个地址没有对应的页面。</p>
              <div className="actions">
                <a className="btn btn-secondary" href="#" onClick={(e) => e.preventDefault()}>
                  全部图片
                </a>
              </div>
            </div>
          </div>
        </main>
      </div>
      {mobile && (
        <nav className="tabbar" aria-label="主导航">
          {NAV.slice(0, 3).map((n) => (
            <button key={n.key} type="button" className="tab">
              <Icon name={n.icon} size={20} />
              <span>{n.short}</span>
            </button>
          ))}
          <button type="button" className="tab">
            <Icon name="more" size={20} />
            <span>更多</span>
          </button>
        </nav>
      )}
    </div>
  );
}

const NAV = [
  { key: 'images', label: '全部图片', short: '图片', icon: 'images', group: '图片' },
  { key: 'trash', label: '回收站', short: '回收站', icon: 'trash', group: '图片' },
  { key: 'tokens', label: '令牌', short: '令牌', icon: 'key', group: '接入' },
  { key: 'settings', label: '设置', short: '设置', icon: 'gear', group: '系统' },
  { key: 'status', label: '状态', short: '状态', icon: 'activity', group: '系统' },
];

const SCREENS = {
  setup: ['empty', 'errors', 'wrong', 'busy', 'done'],
  login: ['empty', 'error', 'offline', 'busy'],
  'not-found': ['default'],
};

function AuthApp() {
  const [screen, setScreen] = React.useState(AUTH_PARAMS.get('s') || 'login');
  const [preset, setPreset] = React.useState(AUTH_PARAMS.get('p') || SCREENS[AUTH_PARAMS.get('s') || 'login'][0]);
  const [theme, setTheme] = React.useState(AUTH_PARAMS.get('theme') || 'light');
  const [open, setOpen] = React.useState(false);
  React.useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);
  const key = `${screen}-${preset}`;
  const opt = (cur, k, label, set) => (
    <button key={k} type="button" aria-pressed={cur === k} onClick={() => set(k)}>
      {label}
    </button>
  );
  return (
    <>
      {screen === 'setup' && <SetupScreen key={key} preset={preset} />}
      {screen === 'login' && <LoginScreen key={key} preset={preset} />}
      {screen === 'not-found' && <NotFoundScreen key={key} />}
      {!AUTH_PARAMS.has('embed') && (
        <div data-proto="controls">
          <button type="button" className="proto-toggle" aria-expanded={open} onClick={() => setOpen(!open)}>
            原型选项
          </button>
          {open && (
            <div className="proto-panel">
              <label>页面</label>
              <div className="opts">
                {opt(screen, 'setup', '初始化', (s) => (setScreen(s), setPreset(SCREENS[s][0])))}
                {opt(screen, 'login', '登录', (s) => (setScreen(s), setPreset(SCREENS[s][0])))}
                {opt(screen, 'not-found', '404', (s) => (setScreen(s), setPreset(SCREENS[s][0])))}
              </div>
              <label>状态</label>
              <div className="opts">{SCREENS[screen].map((p) => opt(preset, p, p, setPreset))}</div>
              <label>主题</label>
              <div className="opts">
                {opt(theme, 'light', '浅色', setTheme)}
                {opt(theme, 'dark', '深色', setTheme)}
              </div>
            </div>
          )}
        </div>
      )}
    </>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<AuthApp />);
