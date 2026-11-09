// 登录 and 初始化.

// After signing in: back to the page that asked for it (?next=/library/...), otherwise the library.
function consoleUrl() {
  const n = new URLSearchParams(location.search).get('next');
  return 'index.html#' + (n && n.startsWith('/') && !n.startsWith('//') ? n : '/library');
}

function AuthShell({ title, children }) {
  return (
    <div className="auth">
      <div className="auth-box">
        <div className="auth-brand">
          <span className="brand-mark" aria-hidden="true" />
          PicBerth
        </div>
        <h1>{title}</h1>
        {children}
      </div>
    </div>
  );
}

function FormError({ text }) {
  if (!text) return null;
  return (
    <div className="alert" role="alert">
      <Icon name="alert" />
      <span>{text}</span>
    </div>
  );
}

function Login({ scenario }) {
  const [user, setUser] = useState('');
  const [pass, setPass] = useState('');
  const [busy, setBusy] = useState(scenario === 'busy');
  const [error, setError] = useState(scenario === 'error' ? '用户名或密码不正确' : scenario === 'offline' ? '无法连接到服务器' : '');
  const submit = (e) => {
    e.preventDefault();
    setBusy(true);
    setError('');
    setTimeout(() => {
      setBusy(false);
      if (pass === 'wrong') setError('用户名或密码不正确');
      else location.href = consoleUrl();
    }, 900);
  };
  return (
    <AuthShell title="登录">
      <form className="auth-form" onSubmit={submit}>
        <div className="field">
          <label htmlFor="l-user">用户名</label>
          <input id="l-user" className="input" autoComplete="username" autoFocus value={user} onChange={(e) => setUser(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="l-pass">密码</label>
          <input id="l-pass" className="input" type="password" autoComplete="current-password" value={pass} onChange={(e) => setPass(e.target.value)} />
        </div>
        <FormError text={error} />
        <button type="submit" className="btn btn-primary btn-block" disabled={busy || !user || !pass}>
          {busy ? '正在登录…' : '登录'}
        </button>
      </form>
    </AuthShell>
  );
}

function Setup({ scenario }) {
  const [v, setV] = useState({ code: '', user: '', pass: '', confirm: '' });
  const [tried, setTried] = useState(scenario === 'validation');
  const [busy, setBusy] = useState(scenario === 'busy');
  const [error, setError] = useState(scenario === 'wrong-code' ? '初始化代码不正确' : scenario === 'done' ? '管理员已经创建，请直接登录' : '');
  const errs = {
    code: !v.code.trim() ? '请输入初始化代码' : v.code.trim().length !== 12 ? '初始化代码为 12 位' : '',
    user: /^[A-Za-z0-9._-]{1,64}$/.test(v.user.trim()) ? '' : '用户名为 1–64 位字母、数字或 . _ -',
    pass: v.pass.length >= 10 && v.pass.length <= 256 ? '' : '密码需要 10–256 个字符',
    confirm: v.confirm === v.pass ? '' : '两次输入的密码不一致',
  };
  const set = (k) => (e) => setV((x) => ({ ...x, [k]: e.target.value }));
  const submit = (e) => {
    e.preventDefault();
    setTried(true);
    if (Object.values(errs).some(Boolean)) return;
    setBusy(true);
    setError('');
    setTimeout(() => (location.href = consoleUrl()), 900);
  };
  const field = (k, id, label, props, hint) => (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <input id={id} className={'input' + (k === 'code' ? ' mono' : '')} aria-invalid={tried && !!errs[k]} value={v[k]} onChange={set(k)} {...props} />
      {tried && errs[k] ? <span className="field-error">{errs[k]}</span> : hint && <span className="s muted">{hint}</span>}
    </div>
  );
  return (
    <AuthShell title="初始化">
      <form className="auth-form" onSubmit={submit} noValidate>
        {field('code', 's-code', '初始化代码', { autoFocus: true, autoComplete: 'off', spellCheck: false, maxLength: 12 }, '在服务器启动日志中')}
        {field('user', 's-user', '用户名', { autoComplete: 'username' })}
        {field('pass', 's-pass', '密码', { type: 'password', autoComplete: 'new-password' })}
        {field('confirm', 's-confirm', '确认密码', { type: 'password', autoComplete: 'new-password' })}
        <FormError text={error} />
        <button type="submit" className="btn btn-primary btn-block" disabled={busy}>
          {busy ? '正在创建…' : '创建管理员'}
        </button>
      </form>
    </AuthShell>
  );
}

function AuthApp() {
  const [themePref, setThemePref] = useTheme();
  const q = new URLSearchParams(location.search);
  const [screen, setScreen] = useState(q.get('screen') || 'login');
  const [scenario, setScenario] = useState(q.get('state') || '');
  const key = screen + scenario;
  const states = {
    login: [
      ['', '默认'],
      ['error', '密码错误'],
      ['offline', '离线'],
      ['busy', '提交中'],
    ],
    setup: [
      ['', '默认'],
      ['validation', '校验失败'],
      ['wrong-code', '代码错误'],
      ['done', '已初始化'],
      ['busy', '提交中'],
    ],
  };
  return (
    <ToastHost>
      {screen === 'login' ? <Login key={key} scenario={scenario} /> : <Setup key={key} scenario={scenario} />}
      <Popover
        trigger={(p) => (
          <button {...p} type="button" className="proto">
            原型
          </button>
        )}
      >
        {(close) => (
          <>
            <div className="menu-group">页面</div>
            <MenuItem checked={screen === 'login'} onClick={() => (setScreen('login'), setScenario(''), close())}>
              登录
            </MenuItem>
            <MenuItem checked={screen === 'setup'} onClick={() => (setScreen('setup'), setScenario(''), close())}>
              初始化
            </MenuItem>
            <div className="menu-sep" />
            <div className="menu-group">状态</div>
            {states[screen].map(([v, l]) => (
              <MenuItem key={v} checked={scenario === v} onClick={() => (setScenario(v), close())}>
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
            <MenuItem icon="image" onClick={() => (location.href = 'index.html#/library')}>
              控制台
            </MenuItem>
          </>
        )}
      </Popover>
    </ToastHost>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<AuthApp />);
