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
      <div className="auth-col">
        <span role="img" aria-label="PicBerth">
          <BrandMark size={20} className="brand-mark-lg" />
        </span>
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

function NotFoundScreen({ theme, setTheme, toast }) {
  const mobile = useMedia('(max-width: 720px)');
  const rail = useMedia('(max-width: 1099px)') && !mobile;
  const [moreOpen, setMoreOpen] = React.useState(false);
  const noop = () => {};
  return (
    <div className={`app ${rail ? 'rail' : ''} ${mobile ? 'mobile' : ''}`} data-screen-label="not-found">
      {!mobile && <Sidebar route={null} go={noop} status={STATUS} theme={theme} setTheme={setTheme} toast={toast} />}
      <div className="workspace">
        <main className="workspace-main">
          <div className="page">
            <div className="empty not-found">
              <h1>页面不存在</h1>
              <a className="btn btn-secondary" href="#" onClick={(e) => e.preventDefault()}>
                全部图片
              </a>
            </div>
          </div>
        </main>
      </div>
      {mobile && <TabBar route={null} go={noop} status={STATUS} moreOpen={moreOpen} onMore={() => setMoreOpen(true)} />}
      {mobile && moreOpen && <MoreSheet route={null} go={noop} status={STATUS} theme={theme} setTheme={setTheme} toast={toast} onClose={() => setMoreOpen(false)} />}
    </div>
  );
}

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
  const [toasts, setToasts] = React.useState([]);
  const toast = (text) => {
    const id = Math.random().toString(36).slice(2);
    setToasts((t) => [...t.slice(-2), { id, text }]);
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 3200);
  };
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
      {screen === 'not-found' && <NotFoundScreen key={key} theme={theme} setTheme={setTheme} toast={toast} />}
      <Toasts toasts={toasts} dismiss={(id) => setToasts((t) => t.filter((x) => x.id !== id))} />
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
