// 令牌, 设置, 状态.

function Head({ title, children }) {
  return (
    <div className="head">
      <div className="head-title">
        <h1>{title}</h1>
      </div>
      {children}
    </div>
  );
}

// ---------------------------------------------------------------- tokens
const EXPIRY = [
  { value: '', label: '永不过期' },
  { value: '30', label: '30 天' },
  { value: '90', label: '90 天' },
  { value: '365', label: '1 年' },
];

function TokenCreate({ onClose, onCreated }) {
  const [name, setName] = useState('');
  const [scopes, setScopes] = useState(['upload']);
  const [exp, setExp] = useState('');
  const [tried, setTried] = useState(false);
  const nameErr = !name.trim() ? '请输入名称' : name.trim().length > 80 ? '名称最多 80 个字符' : '';
  const scopeErr = scopes.length ? '' : '请选择至少一项权限';
  const submit = (e) => {
    e?.preventDefault();
    setTried(true);
    if (nameErr || scopeErr) return;
    onCreated({ name: name.trim(), scopes, expiresAt: exp ? NOW + Number(exp) * DAY : null });
  };
  return (
    <Dialog
      title="新建令牌"
      onClose={onClose}
      footer={
        <>
          <button type="button" className="btn btn-secondary" onClick={onClose}>
            取消
          </button>
          <button type="button" className="btn btn-primary" onClick={submit}>
            创建
          </button>
        </>
      }
    >
      <form className="auth-form" onSubmit={submit}>
        <div className="field">
          <label htmlFor="tk-name">名称</label>
          <input id="tk-name" className="input" value={name} aria-invalid={tried && !!nameErr} onChange={(e) => setName(e.target.value)} />
          {tried && nameErr && <span className="field-error">{nameErr}</span>}
        </div>
        <fieldset className="field" style={{ border: 0, margin: 0, padding: 0 }}>
          <legend className="label medium" style={{ padding: 0, marginBottom: 6 }}>
            权限
          </legend>
          <div className="scope-opts">
            {SCOPES.map((s) => {
              const on = scopes.includes(s.key);
              return (
                <button key={s.key} type="button" className="scope-opt" role="checkbox" aria-checked={on} onClick={() => setScopes((xs) => (on ? xs.filter((x) => x !== s.key) : [...xs, s.key]))}>
                  <span className="check" data-on={String(on)}>{on && <Icon name="check" />}</span>
                  {s.label}
                </button>
              );
            })}
          </div>
          {tried && scopeErr && <span className="field-error">{scopeErr}</span>}
        </fieldset>
        <div className="field">
          <span className="label">有效期</span>
          <Select label="有效期" value={exp} onChange={setExp} options={EXPIRY} width={160} />
        </div>
      </form>
    </Dialog>
  );
}

const CLIENTS = [
  { value: 'picgo', label: 'PicGo' },
  { value: 'sharex', label: 'ShareX' },
  { value: 'curl', label: 'curl' },
];
function TokenReveal({ token, onDone }) {
  const toast = useToast();
  const [client, setClient] = useState('picgo');
  const secret = `${token.prefix}_${fakeId(token.id * 7)}${fakeId(token.id * 13).slice(0, 12)}`;
  const endpoint = 'https://pics.example.com/api/v1/images';
  const snippets = {
    picgo: `API 地址: ${endpoint}\nPOST 参数名: file\nJSON 路径: url\n自定义请求头: {"Authorization":"Bearer ${secret}"}`,
    sharex: `RequestURL: ${endpoint}\nFileFormName: file\nHeaders: Authorization: Bearer ${secret}\nURL: {json:url}`,
    curl: `curl -H "Authorization: Bearer ${secret}" \\\n  -F file=@image.png \\\n  ${endpoint}`,
  };
  return (
    <Dialog
      title="令牌已创建"
      wide
      dismissible={false}
      onClose={onDone}
      footer={
        <button type="button" className="btn btn-primary" data-autofocus onClick={onDone}>
          完成
        </button>
      }
    >
      <div className="notice">
        <Icon name="alert" />
        令牌只显示这一次
      </div>
      <div className="secret">
        <span className="mono">{secret}</span>
        <button type="button" className="btn btn-ghost btn-sm" onClick={() => (copyText(secret), toast('已复制令牌'))}>
          <Icon name="copy" size="s" />
          复制
        </button>
      </div>
      <div className="field">
        <div className="row" style={{ justifyContent: 'space-between' }}>
          <Seg label="客户端" value={client} onChange={setClient} options={CLIENTS} />
          {client === 'sharex' ? (
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => toast('已下载 PicBerth.sxcu')}>
              <Icon name="download" size="s" />
              下载 .sxcu
            </button>
          ) : (
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => (copyText(snippets[client]), toast('已复制'))}>
              <Icon name="copy" size="s" />
              复制
            </button>
          )}
        </div>
        <pre className="snippet">{snippets[client]}</pre>
      </div>
    </Dialog>
  );
}

function Tokens({ dataState, onRetryLoad }) {
  const toast = useToast();
  const [tokens, setTokens] = useState(TOKENS);
  const [creating, setCreating] = useState(false);
  const [revealed, setRevealed] = useState(null);
  const [arming, setArming] = useState(null);
  useEffect(() => {
    if (arming == null) return;
    const t = setTimeout(() => setArming(null), 4000);
    return () => clearTimeout(t);
  }, [arming]);
  const revoke = (t) => {
    if (arming !== t.id) return setArming(t.id);
    setTokens((xs) => xs.map((x) => (x.id === t.id ? { ...x, revokedAt: NOW } : x)));
    setArming(null);
    toast('令牌已吊销');
  };
  const list = dataState === 'empty' ? [] : tokens;
  const live = list.filter((t) => !t.revokedAt);
  const dead = list.filter((t) => t.revokedAt);
  const expiry = (t) => (t.revokedAt ? '已吊销' : t.expiresAt ? (t.expiresAt < NOW ? '已过期' : fmtDate(t.expiresAt).slice(0, 10)) : '永不过期');
  const scopes = (t) => t.scopes.map((s) => SCOPES.find((x) => x.key === s).label).join('、');
  const row = (t) => (
    <div key={t.id} className={'list-row tok-row' + (t.revokedAt ? ' revoked' : '')}>
      <div className="tok-name">
        <span className="n ellipsis">{t.name}</span>
        <span className="mono">{t.prefix}</span>
        <span className="tok-sub">
          <span>{scopes(t)}</span>
          <span>{t.lastUsedAt ? fmtRel(t.lastUsedAt) : '从未使用'}</span>
          <span>{expiry(t)}</span>
        </span>
      </div>
      <span className="c-scope">{scopes(t)}</span>
      <span className="c-used muted num">{t.lastUsedAt ? fmtRel(t.lastUsedAt) : '从未使用'}</span>
      <span className="c-exp muted num">{expiry(t)}</span>
      {t.revokedAt ? (
        <span />
      ) : (
        <button
          type="button"
          className={'btn btn-sm ' + (arming === t.id ? 'btn-danger-armed' : 'btn-ghost')}
          aria-label={arming === t.id ? `确认吊销 ${t.name}` : `吊销 ${t.name}`}
          onClick={() => revoke(t)}
          onBlur={() => arming === t.id && setTimeout(() => setArming((a) => (a === t.id ? null : a)), 150)}
        >
          {arming === t.id ? '确认吊销' : '吊销'}
        </button>
      )}
    </div>
  );
  return (
    <div className="panel-main">
      <Head title="令牌">
        <button type="button" className="btn btn-primary" onClick={() => setCreating(true)}>
          <Icon name="plus" />
          新建令牌
        </button>
      </Head>
      <div className="scroll">
        <div className="page" style={{ maxWidth: 960 }}>
          {dataState === 'loading' ? (
            <div className="list" aria-busy="true">
              {[0, 1, 2].map((i) => (
                <div key={i} className="list-row tok-row">
                  <div className="skeleton" style={{ height: 10, width: '60%' }} />
                </div>
              ))}
            </div>
          ) : dataState === 'error' ? (
            <div className="state" role="alert" style={{ padding: '64px 0' }}>
              <span className="t">令牌加载失败</span>
              <button type="button" className="btn btn-secondary" onClick={onRetryLoad}>
                重试
              </button>
            </div>
          ) : !list.length ? (
            <div className="state" style={{ padding: '64px 0' }}>
              <span className="t">还没有令牌</span>
              <span className="d">PicGo、ShareX 等工具通过令牌上传图片</span>
            </div>
          ) : (
            <div className="list" role="table" aria-label="令牌">
              <div className="list-row list-head tok-row" role="row">
                <span>名称</span>
                <span className="c-scope">权限</span>
                <span className="c-used">最近使用</span>
                <span className="c-exp">到期</span>
                <span />
              </div>
              {live.map(row)}
              {dead.map(row)}
            </div>
          )}
        </div>
      </div>
      {creating && (
        <TokenCreate
          onClose={() => setCreating(false)}
          onCreated={(t) => {
            const id = Math.max(...tokens.map((x) => x.id)) + 1;
            const tok = { ...t, id, prefix: 'pb_' + fakeId(id).slice(0, 5).toUpperCase(), createdAt: NOW, lastUsedAt: null, revokedAt: null };
            setTokens((xs) => [tok, ...xs]);
            setCreating(false);
            setRevealed(tok);
          }}
        />
      )}
      {revealed && <TokenReveal token={revealed} onDone={() => setRevealed(null)} />}
    </div>
  );
}

// ---------------------------------------------------------------- settings
function NumField({ id, value, onChange, unit, min, max, err }) {
  return (
    <>
      <div className="field-wrap has-suffix w-num">
        <input id={id} className="input num" inputMode="numeric" value={value} aria-invalid={!!err} onChange={(e) => onChange(e.target.value.replace(/[^0-9]/g, ''))} />
        <span className="suffix">{unit}</span>
      </div>
      {err && <span className="field-error">{err}</span>}
    </>
  );
}

function Settings({ settings, setSettings, guard }) {
  const toast = useToast();
  const [form, setForm] = useState(settings);
  const dirty = JSON.stringify(form) !== JSON.stringify(settings);
  useEffect(() => {
    guard.current = dirty;
  }, [dirty]);
  const set = (k) => (v) => setForm((f) => ({ ...f, [k]: v }));
  const range = (v, lo, hi) => (v === '' || Number(v) < lo || Number(v) > hi ? `请输入 ${lo} 到 ${hi} 之间的数字` : '');
  const errs = {
    maxUploadMB: range(form.maxUploadMB, 1, settings.hardMaxMB),
    maxPixelsM: range(form.maxPixelsM, 1, 500),
    uploadsPerMinute: range(form.uploadsPerMinute, 1, 6000),
    trashRetentionDays: range(form.trashRetentionDays, 1, 365),
    sessionIdleMinutes: range(form.sessionIdleMinutes, 5, 1440),
    sessionAbsoluteHours: range(form.sessionAbsoluteHours, 1, 720),
    allowedFormats: form.allowedFormats.length ? '' : '至少保留一种格式',
  };
  const invalid = Object.values(errs).some(Boolean);
  const save = () => {
    const n = {};
    for (const [k, v] of Object.entries(form)) n[k] = typeof settings[k] === 'number' ? Number(v) : v;
    setSettings(n);
    setForm(n);
    toast('设置已保存');
  };
  const row = (label, id, ctl, stack) => (
    <div className={'form-row' + (stack ? ' stack' : '')}>
      {id ? (
        <label className="k" htmlFor={id}>
          {label}
        </label>
      ) : (
        <span className="k">{label}</span>
      )}
      <div className="ctl">{ctl}</div>
    </div>
  );
  return (
    <div className="panel-main">
      <Head title="设置" />
      <div className="scroll">
        <div className="page">
          <section className="section">
            <h2>上传</h2>
            <div className="form-rows">
              {row('单文件上限', 's-max', <NumField id="s-max" value={form.maxUploadMB} onChange={set('maxUploadMB')} unit="MB" err={errs.maxUploadMB} />)}
              {row('像素上限', 's-px', <NumField id="s-px" value={form.maxPixelsM} onChange={set('maxPixelsM')} unit="MP" err={errs.maxPixelsM} />)}
              {row(
                '格式',
                null,
                <>
                  {Object.keys(FORMAT_LABEL).map((k) => (
                    <button
                      key={k}
                      type="button"
                      className="chip"
                      aria-pressed={form.allowedFormats.includes(k)}
                      onClick={() => set('allowedFormats')(form.allowedFormats.includes(k) ? form.allowedFormats.filter((x) => x !== k) : [...form.allowedFormats, k])}
                    >
                      {form.allowedFormats.includes(k) && <Icon name="check" size="s" />}
                      {FORMAT_LABEL[k]}
                    </button>
                  ))}
                  {errs.allowedFormats && <span className="field-error">{errs.allowedFormats}</span>}
                </>,
                true,
              )}
              {row('频率上限', 's-rate', <NumField id="s-rate" value={form.uploadsPerMinute} onChange={set('uploadsPerMinute')} unit="次/分" err={errs.uploadsPerMinute} />)}
              {row('重复检测', null, <Switch on={form.dedupe} onChange={set('dedupe')} label="重复检测" />)}
            </div>
          </section>
          <section className="section">
            <h2>回收站</h2>
            <div className="form-rows">
              {row('保留时间', 's-trash', <NumField id="s-trash" value={form.trashRetentionDays} onChange={set('trashRetentionDays')} unit="天" err={errs.trashRetentionDays} />)}
            </div>
          </section>
          <section className="section">
            <h2>会话</h2>
            <div className="form-rows">
              {row('空闲超时', 's-idle', <NumField id="s-idle" value={form.sessionIdleMinutes} onChange={set('sessionIdleMinutes')} unit="分钟" err={errs.sessionIdleMinutes} />)}
              {row('最长有效', 's-abs', <NumField id="s-abs" value={form.sessionAbsoluteHours} onChange={set('sessionAbsoluteHours')} unit="小时" err={errs.sessionAbsoluteHours} />)}
            </div>
          </section>
          <section className="section">
            <h2>地址</h2>
            <div className="form-rows">
              {row('公开地址', null, <span className="mono ellipsis" style={{ fontSize: 'var(--fs-m)' }}>{form.publicUrl}</span>)}
              {row('图片地址', null, <span className="mono ellipsis" style={{ fontSize: 'var(--fs-m)' }}>{form.imageUrl}</span>)}
            </div>
          </section>
        </div>
      </div>
      {dirty && (
        <div className="save-bar" role="region" aria-label="未保存的更改">
          <span className="t">有未保存的更改</span>
          <button type="button" className="btn btn-ghost" onClick={() => setForm(settings)}>
            放弃
          </button>
          <button type="button" className="btn btn-primary" disabled={invalid} onClick={save}>
            保存
          </button>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------- status
function Status({ dataState, onRetryLoad }) {
  const toast = useToast();
  const [running, setRunning] = useState(null);
  const [results, setResults] = useState({
    quick: { tone: 'success', text: `正常`, at: STATUS.lastQuick.at },
    full: { tone: 'success', text: `正常`, at: STATUS.lastFull.at },
    snapshot: { tone: 'warning', text: '', at: STATUS.lastSnapshot },
  });
  const [warnings, setWarnings] = useState(STATUS.warnings);
  const [audit, setAudit] = useState(AUDIT.slice(0, 6));
  const run = (k) => {
    setRunning(k);
    setTimeout(() => {
      setRunning(null);
      setResults((r) => ({ ...r, [k]: { tone: 'success', text: k === 'snapshot' ? '已创建' : '正常', at: NOW } }));
      if (k === 'snapshot') setWarnings((w) => w.filter((x) => x.code !== 'snapshot-stale'));
      toast(k === 'snapshot' ? '快照已创建' : k === 'quick' ? '快速检查完成' : '完整检查完成');
    }, 1400);
  };
  const s = STATUS;
  const used = 1 - s.disk.free / s.disk.total;
  if (dataState === 'loading' || dataState === 'error')
    return (
      <div className="panel-main">
        <Head title="状态" />
        <div className="scroll">
          {dataState === 'error' ? (
            <div className="state" role="alert">
              <span className="t">状态加载失败</span>
              <button type="button" className="btn btn-secondary" onClick={onRetryLoad}>
                重试
              </button>
            </div>
          ) : (
            <div className="page" aria-busy="true">
              <div className="skeleton" style={{ height: 76, borderRadius: 8 }} />
            </div>
          )}
        </div>
      </div>
    );
  const maint = [
    { k: 'quick', name: '快速检查', btn: '检查' },
    { k: 'full', name: '完整检查', btn: '检查' },
    { k: 'snapshot', name: '快照', btn: '创建' },
  ];
  return (
    <div className="panel-main">
      <Head title="状态">
        <button type="button" className="btn btn-ghost" onClick={() => toast('状态已刷新')}>
          <Icon name="refresh" />
          刷新
        </button>
      </Head>
      <div className="scroll">
        <div className="page" style={{ maxWidth: 960 }}>
          {warnings.length > 0 && (
            <section className="section" aria-label="警告">
              <div>
                {warnings.map((w) => (
                  <div key={w.code} className="warn-row" role="alert">
                    <span className="dot dot-warning" />
                    <span className="t">{w.title}</span>
                    <span className="d">{w.help}</span>
                  </div>
                ))}
              </div>
            </section>
          )}

          <div className="figures">
            <div className="figure">
              <span className="v">{s.images.active.toLocaleString('zh-CN')}</span>
              <span className="k">图片</span>
            </div>
            <div className="figure">
              <span className="v">{fmtBytes(s.images.bytes)}</span>
              <span className="k">占用空间</span>
            </div>
            <div className="figure">
              <span className="v">{s.images.trashed}</span>
              <span className="k">回收站</span>
            </div>
            <div className="figure">
              <span className="v">{fmtBytes(s.dbBytes)}</span>
              <span className="k">数据库</span>
            </div>
            <div className="figure">
              <span className="v">{fmtBytes(s.disk.free)}</span>
              <span className="k">磁盘剩余</span>
              <div className="meter" role="progressbar" aria-label="磁盘已用" aria-valuenow={Math.round(used * 100)} aria-valuemin={0} aria-valuemax={100}>
                <i style={{ width: `${used * 100}%` }} />
              </div>
            </div>
          </div>

          <section className="section">
            <h2>维护</h2>
            <div className="list">
              {maint.map((m) => {
                const r = results[m.k];
                return (
                  <div key={m.k} className="list-row maint-row">
                    <span className="medium">{m.name}</span>
                    <span className="c-res badge">
                      <span className={'dot dot-' + r.tone} />
                      {r.text && <span>{r.text}</span>}
                      <span className="num">{fmtDate(r.at)}</span>
                    </span>
                    <button type="button" className="btn btn-secondary btn-sm" disabled={running != null} onClick={() => run(m.k)}>
                      {running === m.k ? (m.k === 'snapshot' ? '创建中…' : '检查中…') : m.btn}
                    </button>
                  </div>
                );
              })}
            </div>
          </section>

          <section className="section">
            <h2>操作记录</h2>
            <div className="list" role="table" aria-label="操作记录">
              <div className="list-row list-head audit-row" role="row">
                <span>时间</span>
                <span>操作者</span>
                <span>操作</span>
                <span>对象</span>
              </div>
              {audit.map((a, i) => (
                <div key={i} className="list-row audit-row" role="row">
                  <span className="c-at muted num">{fmtDate(a.at)}</span>
                  <span className="c-actor ellipsis">{a.actor}</span>
                  <span className="c-action medium">{a.action}</span>
                  <span className="c-target ellipsis muted">{a.target}</span>
                </div>
              ))}
            </div>
            {audit.length < AUDIT.length && (
              <div>
                <button type="button" className="btn btn-ghost" style={{ marginLeft: -12 }} onClick={() => setAudit(AUDIT)}>
                  加载更多
                </button>
              </div>
            )}
          </section>

          <section className="section">
            <h2>运行信息</h2>
            <dl className="dl">
              <dt>版本</dt>
              <dd className="num row" style={{ gap: 12 }}>
                <span>{s.version}</span>
                <span className="mono muted">{s.commit}</span>
              </dd>
              <dt>运行环境</dt>
              <dd className="num row" style={{ gap: 12 }}>
                <span>{s.goVersion}</span>
                <span>SQLite {s.sqliteVersion}</span>
              </dd>
              <dt>已运行</dt>
              <dd>{s.uptime}</dd>
              <dt>数据目录</dt>
              <dd className="mono" style={{ fontSize: 'var(--fs-m)' }}>
                {s.dataDir}
              </dd>
            </dl>
          </section>
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { Tokens, Settings, Status });
