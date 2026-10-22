// Admin pages: tokens, settings, status. Page content only; the shell is provided elsewhere.

(function admInjectStyle() {
  if (document.getElementById('adm-style')) return;
  const el = document.createElement('style');
  el.id = 'adm-style';
  el.textContent = `
.adm-stack { display: flex; flex-direction: column; gap: 16px; }
.adm-table tbody tr { cursor: default; }
.adm-table td { height: 56px; }
.adm-revoked td { color: var(--subtle); }
.adm-revoked .cell-name .name { font-weight: 400; }
.adm-tcards { display: flex; flex-direction: column; gap: 8px; }
.adm-tcard { display: flex; flex-direction: column; gap: 10px; padding: 14px; }
.adm-tcard.adm-revoked { color: var(--subtle); }
.adm-tcard-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.adm-kv { display: flex; justify-content: space-between; gap: 12px; font-size: 13px; }
.adm-kv > span:first-child { color: var(--subtle); }
.adm-badges { display: flex; flex-wrap: wrap; gap: 4px; }
.adm-form { display: flex; flex-direction: column; gap: 18px; }
.adm-opts { display: flex; flex-direction: column; gap: 4px; }
.adm-opts-row { display: flex; flex-wrap: wrap; gap: 8px 16px; }
.adm-opt { display: flex; align-items: flex-start; gap: 10px; padding: 8px 10px; margin: 0 -10px; border-radius: var(--r); cursor: pointer; transition: background-color var(--d1) var(--ease-out); }
.adm-opt:hover { background: var(--tint); }
.adm-opt .check { margin-top: 2px; flex: none; }
.adm-opt strong { display: block; font-size: 13px; font-weight: 500; }
.adm-opt-inline { display: inline-flex; align-items: center; gap: 8px; min-height: var(--h); font-size: 13px; cursor: pointer; }
.adm-invalid, .adm-invalid:hover { border-color: var(--danger); }
.adm-invalid:focus { border-color: var(--danger); box-shadow: 0 0 0 3px var(--danger-soft); }
.adm-pre { margin: 0; padding: 12px; border: 1px solid var(--hairline); border-radius: var(--r-lg); background: var(--tint); font-family: var(--mono); font-size: 12px; line-height: 1.6; white-space: pre-wrap; overflow-wrap: anywhere; }
.adm-secret { display: flex; gap: 6px; }
.adm-secret .input { flex: 1; font-family: var(--mono); font-size: 12px; }
.adm-row { display: grid; grid-template-columns: minmax(180px, 240px) minmax(0, 1fr); gap: 8px 32px; align-items: start; padding: 16px 20px; border-bottom: 1px solid var(--hairline); }
.adm-row:last-child { border-bottom: 0; }
.adm-row > div:first-child { padding-top: calc((var(--h) - 20px) / 2); }
.adm-row .label { display: block; font-size: 13px; font-weight: 500; line-height: 20px; }
.adm-row .hint { margin-top: 2px; }
.adm-ctl { display: flex; flex-direction: column; align-items: flex-start; gap: 6px; min-width: 0; }
.adm-inline { display: flex; align-items: center; gap: 8px; }
.adm-value { display: flex; align-items: center; min-height: var(--h); min-width: 0; overflow-wrap: anywhere; }
.adm-sticky { position: sticky; bottom: calc(16px + var(--bottom-ui, 0px)); z-index: 30; display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: 8px; margin-top: 16px; padding: 10px 10px 10px 16px; border: 1px solid var(--hairline); border-radius: var(--r-lg); background: var(--raised); box-shadow: var(--shadow-float); animation: bar-in var(--d2) var(--ease-out); }
.adm-sticky .grow { flex: 1; min-width: 140px; font-size: 13px; color: var(--text); font-weight: 500; }
.adm-sticky .grow::before { content: ''; display: inline-block; width: 6px; height: 6px; margin: 0 8px 2px 0; border-radius: 50%; background: var(--warning); vertical-align: middle; }
.adm-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); overflow: hidden; }
.adm-metric { display: flex; flex-direction: column; gap: 4px; min-width: 0; padding: 16px 20px; box-shadow: -1px 0 0 var(--hairline), 0 -1px 0 var(--hairline); }
.adm-metric .k { font-size: 12px; color: var(--subtle); }
.adm-metric .v { font-size: 22px; font-weight: 600; line-height: 1.25; font-variant-numeric: tabular-nums; color: var(--strong); letter-spacing: -0.01em; }
.adm-metric .s { font-size: 12px; color: var(--subtle); }
.adm-metric .progress { margin: 6px 0 2px; }
.adm-result { display: flex; flex-wrap: wrap; align-items: center; gap: 4px 8px; font-size: 13px; overflow-wrap: anywhere; }
.adm-result.subtle { color: var(--subtle); }
.adm-run { width: min(260px, 100%); }
.adm-audit-list { display: flex; flex-direction: column; }
.adm-audit-item { display: flex; flex-direction: column; gap: 2px; padding: 10px 16px; border-bottom: 1px solid var(--hairline); font-size: 13px; }
.adm-audit-item:last-child { border-bottom: 0; }
@media (max-width: 900px) { .adm-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 720px) { .adm-row { grid-template-columns: minmax(0, 1fr); padding: 14px 16px; } .adm-row > div:first-child { padding-top: 0; } .card-head { padding: 12px 16px; } .adm-sticky .btn { height: var(--h-touch); } .adm-metric { padding: 14px 16px; } .adm-metric .v { font-size: 20px; } }
`;
  document.head.appendChild(el);
})();

// ---------------------------------------------------------------- tokens

const ADM_API_URL = 'https://pics.example.com/api/v1/images';
const ADM_SCOPE_LABEL = { upload: '上传', read: '只读', manage: '管理' };
const ADM_SCOPE_OPTIONS = [
  { key: 'upload', hint: '上传图片，只能查看自己上传的图片' },
  { key: 'read', hint: '查看全部图片' },
  { key: 'manage', hint: '修改可见性、删除和恢复图片' },
];
const ADM_EXPIRY_OPTIONS = [
  { value: '0', label: '永不过期' },
  { value: '30', label: '30 天' },
  { value: '90', label: '90 天' },
  { value: '365', label: '1 年' },
];
const ADM_B32 = 'abcdefghijklmnopqrstuvwxyz234567';

function admNewSecret() {
  const bytes = new Uint8Array(32);
  if (window.crypto && window.crypto.getRandomValues) window.crypto.getRandomValues(bytes);
  else for (let i = 0; i < 32; i++) bytes[i] = Math.floor(Math.random() * 256);
  let out = 'pb_';
  for (let i = 0; i < 32; i++) out += ADM_B32[bytes[i] % 32];
  return out;
}

function admSnippet(kind, secret) {
  if (kind === 'curl') {
    return `curl -H "Authorization: Bearer ${secret}" -F file=@shot.png ${ADM_API_URL}`;
  }
  if (kind === 'picgo') {
    return [`API 地址: ${ADM_API_URL}`, 'POST 参数名: file', 'JSON 路径: url', `自定义请求头: {"Authorization":"Bearer ${secret}"}`].join('\n');
  }
  return [`RequestURL: ${ADM_API_URL}`, 'FileFormName: file', 'URL: {json:url}', `Headers: Authorization: Bearer ${secret}`].join('\n');
}

function AdmTokenModal({ onClose, onDone, toast }) {
  const [name, setName] = React.useState('');
  const [scopes, setScopes] = React.useState({ upload: true, read: false, manage: false });
  const [days, setDays] = React.useState('0');
  const [tried, setTried] = React.useState(false);
  const [created, setCreated] = React.useState(null);
  const [kind, setKind] = React.useState('curl');

  const nameErr = name.trim() ? '' : '请输入名称';
  const scopeErr = Object.values(scopes).some(Boolean) ? '' : '请选择至少一项权限';

  const submit = () => {
    setTried(true);
    if (nameErr || scopeErr) return;
    const secret = admNewSecret();
    const d = Number(days);
    setCreated({
      secret,
      token: {
        id: Date.now(),
        name: name.trim(),
        prefix: 'pb_' + secret.slice(3, 8).toUpperCase(),
        scopes: ['upload', 'read', 'manage'].filter((k) => scopes[k]),
        createdAt: NOW,
        lastUsedAt: null,
        expiresAt: d ? NOW + d * DAY : null,
        revokedAt: null,
      },
    });
  };

  if (created) {
    const finish = () => onDone(created.token);
    return (
      <Modal
        title="令牌已创建"
        width="520px"
        onClose={finish}
        footer={
          <button type="button" className="btn btn-primary" onClick={finish}>
            完成
          </button>
        }
      >
        <div className="adm-form">
          <div className="alert warning">
            <Icon name="alert" />
            <div className="grow">关闭后将无法再次查看此令牌</div>
          </div>
          <div className="field">
            <label htmlFor="adm-secret">令牌</label>
            <div className="adm-secret">
              <input id="adm-secret" className="input" readOnly value={created.secret} onFocus={(e) => e.target.select()} />
              <CopyButton text={created.secret} />
            </div>
          </div>
          <div className="adm-form" style={{ gap: 10 }}>
            <div>
              <Seg
                label="使用方式"
                value={kind}
                onChange={setKind}
                options={[
                  { key: 'curl', label: 'curl' },
                  { key: 'picgo', label: 'PicGo' },
                  { key: 'sharex', label: 'ShareX' },
                ]}
              />
            </div>
            {kind === 'sharex' && (
              <div>
                <button type="button" className="btn btn-secondary btn-sm" onClick={() => toast('已下载 PicBerth.sxcu')}>
                  <Icon name="download" />
                  下载 .sxcu
                </button>
              </div>
            )}
            <pre className="adm-pre">{admSnippet(kind, created.secret)}</pre>
          </div>
        </div>
      </Modal>
    );
  }

  return (
    <Modal
      title="新建令牌"
      width="520px"
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
      <div className="adm-form">
        <div className="field">
          <label htmlFor="adm-tk-name">名称</label>
          <input
            id="adm-tk-name"
            className={`input ${tried && nameErr ? 'adm-invalid' : ''}`}
            aria-invalid={tried && nameErr ? 'true' : undefined}
            value={name}
            autoFocus
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && submit()}
          />
          {tried && nameErr && <span className="error-text">{nameErr}</span>}
        </div>
        <div className="field">
          <span style={{ fontSize: 13, fontWeight: 500 }}>权限</span>
          <div className="adm-opts">
            {ADM_SCOPE_OPTIONS.map((o) => (
              <label key={o.key} className="adm-opt">
                <input type="checkbox" className="check" checked={scopes[o.key]} onChange={(e) => setScopes({ ...scopes, [o.key]: e.target.checked })} />
                <span>
                  <strong>{ADM_SCOPE_LABEL[o.key]}</strong>
                  <span className="hint">{o.hint}</span>
                </span>
              </label>
            ))}
          </div>
          {tried && scopeErr && <span className="error-text">{scopeErr}</span>}
        </div>
        <div className="field">
          <label htmlFor="adm-tk-exp">有效期</label>
          <select id="adm-tk-exp" className="select" value={days} onChange={(e) => setDays(e.target.value)}>
            {ADM_EXPIRY_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
      </div>
    </Modal>
  );
}

function AdmExpiry({ t }) {
  if (t.revokedAt) return <span className="badge">已吊销</span>;
  if (t.expiresAt == null) return <span>永不过期</span>;
  if (t.expiresAt < NOW) return <span className="badge badge-danger">已过期</span>;
  return <span className="num">{fmtDate(t.expiresAt).slice(0, 10)}</span>;
}

function AdmScopeBadges({ scopes }) {
  return (
    <span className="adm-badges">
      {scopes.map((s) => (
        <span key={s} className={`badge ${s === 'manage' ? 'badge-warning' : ''}`}>
          {ADM_SCOPE_LABEL[s]}
        </span>
      ))}
    </span>
  );
}

function TokensPage({ tokens, setTokens, toast }) {
  const mobile = useMedia('(max-width: 720px)');
  const [creating, setCreating] = React.useState(false);
  const [revoking, setRevoking] = React.useState(null);

  const revoke = () => {
    const id = revoking.id;
    setTokens(tokens.map((t) => (t.id === id ? { ...t, revokedAt: NOW } : t)));
    setRevoking(null);
    toast('令牌已吊销');
  };
  const added = (token) => {
    setTokens([token, ...tokens]);
    setCreating(false);
    toast('令牌已创建');
  };

  const lastUsed = (t) => (t.lastUsedAt ? fmtRel(t.lastUsedAt) : <span className="subtle">从未使用</span>);
  const newBtn = (
    <button type="button" className="btn btn-primary" onClick={() => setCreating(true)}>
      <Icon name="plus" />
      新建令牌
    </button>
  );

  return (
    <div className="page" style={{ '--page-w': '1040px' }}>
      <div className="page-head">
        <div>
          <h1>令牌</h1>
          <p>用于 PicGo、ShareX 和脚本上传</p>
        </div>
        {tokens.length > 0 && <div className="page-actions">{newBtn}</div>}
      </div>

      {tokens.length === 0 ? (
        <div className="empty">
          <strong>还没有令牌</strong>
          <p>新建令牌后，可以在 PicGo、ShareX 或脚本中上传图片。</p>
          <div className="actions">{newBtn}</div>
        </div>
      ) : mobile ? (
        <div className="adm-tcards">
          {tokens.map((t) => (
            <div key={t.id} className={`card adm-tcard ${t.revokedAt ? 'adm-revoked' : ''}`}>
              <div className="adm-tcard-top">
                <div className="cell-name">
                  <span className="name truncate">{t.name}</span>
                  <span className="sub mono">{t.prefix}…</span>
                </div>
                {t.revokedAt ? (
                  <span className="badge">已吊销</span>
                ) : (
                  <button type="button" className="btn btn-danger btn-sm" onClick={() => setRevoking(t)}>
                    吊销
                  </button>
                )}
              </div>
              <AdmScopeBadges scopes={t.scopes} />
              <div className="adm-kv">
                <span>最近使用</span>
                <span>{lastUsed(t)}</span>
              </div>
              {!t.revokedAt && (
                <div className="adm-kv">
                  <span>过期</span>
                  <span>
                    <AdmExpiry t={t} />
                  </span>
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="table-wrap">
          <table className="table adm-table">
            <thead>
              <tr>
                <th>名称</th>
                <th>权限</th>
                <th>最近使用</th>
                <th>过期</th>
                <th className="c-actions" />
              </tr>
            </thead>
            <tbody>
              {tokens.map((t) => (
                <tr key={t.id} className={t.revokedAt ? 'adm-revoked' : ''}>
                  <td>
                    <div className="cell-name">
                      <span className="name truncate">{t.name}</span>
                      <span className="sub mono">{t.prefix}…</span>
                    </div>
                  </td>
                  <td>
                    <AdmScopeBadges scopes={t.scopes} />
                  </td>
                  <td>{lastUsed(t)}</td>
                  <td>
                    <AdmExpiry t={t} />
                  </td>
                  <td className="c-actions">
                    {!t.revokedAt && (
                      <button type="button" className="btn btn-danger btn-sm" onClick={() => setRevoking(t)}>
                        吊销
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {creating && <AdmTokenModal onClose={() => setCreating(false)} onDone={added} toast={toast} />}
      {revoking && (
        <Modal
          title={`吊销令牌「${revoking.name}」？`}
          width="440px"
          onClose={() => setRevoking(null)}
          footer={
            <>
              <button type="button" className="btn btn-secondary" onClick={() => setRevoking(null)}>
                取消
              </button>
              <button type="button" className="btn btn-danger-solid" onClick={revoke}>
                吊销
              </button>
            </>
          }
        >
          <p>使用此令牌的工具将无法继续上传。</p>
        </Modal>
      )}
    </div>
  );
}

// ---------------------------------------------------------------- settings

const ADM_FORMAT_ORDER = ['jpeg', 'png', 'gif', 'webp', 'avif'];
const ADM_NUM_FIELDS = ['maxUploadMB', 'maxPixelsM', 'trashRetentionDays', 'uploadsPerMinute', 'sessionIdleMinutes', 'sessionAbsoluteHours'];

function admToDraft(s) {
  const d = { ...s, allowedFormats: [...s.allowedFormats] };
  ADM_NUM_FIELDS.forEach((k) => {
    d[k] = String(s[k]);
  });
  return d;
}

function admFromDraft(d) {
  const s = { ...d, allowedFormats: ADM_FORMAT_ORDER.filter((f) => d.allowedFormats.includes(f)) };
  ADM_NUM_FIELDS.forEach((k) => {
    s[k] = Number(d[k]);
  });
  return s;
}

function admRange(v, min, max) {
  const t = String(v).trim();
  if (!/^\d+$/.test(t)) return `请输入 ${min} 到 ${max} 之间的数字`;
  const n = Number(t);
  return n < min || n > max ? `请输入 ${min} 到 ${max} 之间的数字` : '';
}

function admValidate(d, hardMax) {
  const e = {};
  e.maxUploadMB = admRange(d.maxUploadMB, 1, hardMax);
  e.maxPixelsM = admRange(d.maxPixelsM, 1, 1000);
  e.trashRetentionDays = admRange(d.trashRetentionDays, 0, 3650);
  e.uploadsPerMinute = admRange(d.uploadsPerMinute, 1, 6000);
  e.sessionIdleMinutes = admRange(d.sessionIdleMinutes, 1, 1440);
  e.sessionAbsoluteHours = admRange(d.sessionAbsoluteHours, 1, 720);
  e.allowedFormats = d.allowedFormats.length ? '' : '至少选择一种格式';
  return e;
}

function AdmRow({ label, hint, htmlFor, children }) {
  return (
    <div className="adm-row">
      <div>
        {htmlFor ? (
          <label className="label" htmlFor={htmlFor}>
            {label}
          </label>
        ) : (
          <span className="label">{label}</span>
        )}
        {hint && <div className="hint">{hint}</div>}
      </div>
      <div className="adm-ctl">{children}</div>
    </div>
  );
}

function AdmCard({ title, children }) {
  return (
    <section className="card">
      <div className="card-head">
        <h2>{title}</h2>
      </div>
      <div>{children}</div>
    </section>
  );
}

function SettingsPage({ settings, setSettings, toast }) {
  const [draft, setDraft] = React.useState(() => admToDraft(settings));
  const base = admToDraft(settings);
  const hardMax = settings.hardMaxMB || 100;
  const errors = admValidate(draft, hardMax);
  const invalid = Object.values(errors).some(Boolean);

  const same = (k) => (k === 'allowedFormats' ? ADM_FORMAT_ORDER.filter((f) => draft[k].includes(f)).join() === ADM_FORMAT_ORDER.filter((f) => base[k].includes(f)).join() : String(draft[k]).trim() === String(base[k]));
  const changed = ['maxUploadMB', 'maxPixelsM', 'allowedFormats', 'defaultVisibility', 'dedupe', 'trashRetentionDays', 'uploadsPerMinute', 'sessionIdleMinutes', 'sessionAbsoluteHours'].filter((k) => !same(k));
  const dirty = changed.length > 0;

  const set = (k, v) => setDraft((d) => ({ ...d, [k]: v }));
  const toggleFormat = (f, on) => set('allowedFormats', on ? [...draft.allowedFormats, f] : draft.allowedFormats.filter((x) => x !== f));
  const save = () => {
    if (invalid) return;
    const next = admFromDraft(draft);
    setSettings(next);
    setDraft(admToDraft(next));
    toast('设置已保存');
  };
  const discard = () => setDraft(admToDraft(settings));

  const num = (key, id, unit, ariaLabel) => (
    <>
      <div className="adm-inline">
        <input
          id={id}
          className={`input num ${errors[key] ? 'adm-invalid' : ''}`}
          style={{ width: 112 }}
          inputMode="numeric"
          aria-invalid={errors[key] ? 'true' : undefined}
          aria-label={ariaLabel}
          value={draft[key]}
          onChange={(e) => set(key, e.target.value)}
        />
        {unit && <span className="subtle">{unit}</span>}
      </div>
      {errors[key] && <span className="error-text">{errors[key]}</span>}
    </>
  );

  return (
    <div className="page" style={{ '--page-w': '1040px' }}>
      <div className="page-head">
        <div>
          <h1>设置</h1>
        </div>
      </div>

      <div className="adm-stack">
        <AdmCard title="上传">
          <AdmRow label="单文件上限" hint={`不超过 ${hardMax} MB`} htmlFor="adm-max-mb">
            {num('maxUploadMB', 'adm-max-mb', 'MB')}
          </AdmRow>
          <AdmRow label="像素上限" htmlFor="adm-max-px">
            {num('maxPixelsM', 'adm-max-px', '百万像素')}
          </AdmRow>
          <AdmRow label="允许的格式">
            <div className="adm-opts-row">
              {ADM_FORMAT_ORDER.map((f) => (
                <label key={f} className="adm-opt-inline">
                  <input type="checkbox" className="check" checked={draft.allowedFormats.includes(f)} onChange={(e) => toggleFormat(f, e.target.checked)} />
                  {FORMAT_LABEL[f]}
                </label>
              ))}
            </div>
            {errors.allowedFormats && <span className="error-text">{errors.allowedFormats}</span>}
          </AdmRow>
          <AdmRow label="默认可见性">
            <Seg
              label="默认可见性"
              value={draft.defaultVisibility}
              onChange={(v) => set('defaultVisibility', v)}
              options={[
                { key: 'public', label: '公开' },
                { key: 'private', label: '私有' },
              ]}
            />
          </AdmRow>
          <AdmRow label="重复检测" hint="相同内容的图片只保存一份，再次上传时返回原链接">
            <div className="adm-value">
              <Switch checked={draft.dedupe} onChange={(v) => set('dedupe', v)} label="重复检测" />
            </div>
          </AdmRow>
        </AdmCard>

        <AdmCard title="回收站">
          <AdmRow label="保留天数" hint="0 表示删除后立即永久清除" htmlFor="adm-trash-days">
            {num('trashRetentionDays', 'adm-trash-days', '天')}
          </AdmRow>
        </AdmCard>

        <AdmCard title="频率限制">
          <AdmRow label="每分钟上传次数" htmlFor="adm-rate">
            {num('uploadsPerMinute', 'adm-rate', '次')}
          </AdmRow>
        </AdmCard>

        <AdmCard title="会话">
          <AdmRow label="空闲超时" htmlFor="adm-idle">
            {num('sessionIdleMinutes', 'adm-idle', '分钟')}
          </AdmRow>
          <AdmRow label="最长有效" htmlFor="adm-abs">
            {num('sessionAbsoluteHours', 'adm-abs', '小时')}
          </AdmRow>
        </AdmCard>

        <AdmCard title="地址">
          <AdmRow label="公开地址" hint="在配置文件或环境变量中修改">
            <div className="adm-value mono">{settings.publicUrl}</div>
          </AdmRow>
          <AdmRow label="图片地址" hint="在配置文件或环境变量中修改">
            <div className="adm-value mono">{settings.imageUrl}</div>
          </AdmRow>
        </AdmCard>
      </div>

      {dirty && (
        <div className="adm-sticky">
          <span className="grow">有 {changed.length} 项未保存的更改</span>
          <button type="button" className="btn btn-secondary" onClick={discard}>
            放弃
          </button>
          <button type="button" className="btn btn-primary" onClick={save} disabled={invalid}>
            保存
          </button>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------- status

const ADM_GB = 1073741824;

function AdmMetric({ label, value, sub, children }) {
  return (
    <div className="adm-metric">
      <span className="k">{label}</span>
      <span className="v">{value}</span>
      {children}
      {sub && <span className="s">{sub}</span>}
    </div>
  );
}

function AdmRunRow({ label, hint, idleLabel, busyLabel, running, disabled, onRun, result }) {
  return (
    <AdmRow label={label} hint={hint}>
      <button type="button" className="btn btn-secondary" disabled={running || disabled} onClick={onRun}>
        {running ? busyLabel : idleLabel}
      </button>
      {running && (
        <div className="progress indeterminate adm-run" role="progressbar" aria-label={label}>
          <span />
        </div>
      )}
      {!running && result && (
        <div className={`adm-result ${result.subtle ? 'subtle' : ''}`}>
          {result.badge && <span className={`badge ${result.tone || 'badge-success'}`}>{result.badge}</span>}
          {result.body}
        </div>
      )}
    </AdmRow>
  );
}

function StatusPage({ status, images, toast }) {
  const mobile = useMedia('(max-width: 720px)');
  const timers = React.useRef([]);
  React.useEffect(() => () => timers.current.forEach(clearTimeout), []);

  const active = images.filter((i) => i.state === 'active');
  const trashed = images.filter((i) => i.state === 'trashed');
  const totalBytes = images.reduce((s, i) => s + i.size, 0);
  const gpsCount = active.filter((i) => i.hasGps).length;
  const { disk, db } = status;
  const usedShare = disk.totalBytes ? Math.round(((disk.totalBytes - disk.freeBytes) / disk.totalBytes) * 100) : 0;
  const days = Math.floor(status.uptimeSeconds / 86400);
  const uptime = days > 0 ? `${days} 天` : `${Math.max(1, Math.floor(status.uptimeSeconds / 3600))} 小时`;

  const [running, setRunning] = React.useState(null);
  const [snapDone, setSnapDone] = React.useState(false);
  const [auditExtra, setAuditExtra] = React.useState([]);
  const lc = status.lastCheck;
  const ls = status.lastSnapshot;
  const lastCheckResult = lc
    ? {
        badge: lc.ok ? '正常' : `发现 ${lc.issues} 个问题`,
        tone: lc.ok ? 'badge-success' : 'badge-danger',
        body: <span className="subtle">上次检查 {fmtRel(lc.at)}</span>,
      }
    : null;
  const [results, setResults] = React.useState({
    quick: lc && lc.mode === 'quick' ? lastCheckResult : null,
    full: lc && lc.mode === 'full' ? lastCheckResult : null,
    snapshot: ls
      ? {
          subtle: true,
          body: (
            <>
              <span>上次快照 {fmtRel(ls.at)}</span>
              <span className="mono">{ls.path}</span>
            </>
          ),
        }
      : null,
  });

  const run = (key, ms, finish) => {
    if (running) return;
    setRunning(key);
    const t = setTimeout(() => {
      setRunning(null);
      finish();
    }, ms);
    timers.current.push(t);
  };
  const runCheck = (key, seconds, label) =>
    run(key, 1500, () => {
      setResults((r) => ({
        ...r,
        [key]: { badge: '正常', tone: 'badge-success', body: <span>检查了 {active.length} 张图片，用时 {seconds} 秒</span> },
      }));
      toast(`${label}完成`);
    });
  const runSnapshot = () =>
    run('snapshot', 1500, () => {
      const path = '/var/lib/picberth/snapshots/20261021T073000Z';
      setResults((r) => ({
        ...r,
        snapshot: {
          badge: '已创建',
          tone: 'badge-success',
          body: (
            <>
              <span className="mono">{path}</span>
              <span>· 2.3 MB</span>
            </>
          ),
        },
      }));
      setSnapDone(true);
      setAuditExtra((a) => [{ at: NOW, actor: 'admin', action: '创建快照', target: '20261021T073000Z' }, ...a]);
      toast('快照已创建');
    });

  const warnings = (status.warnings || []).filter((w) => !(snapDone && w.code === 'snapshot-stale'));
  const audit = [...auditExtra, ...AUDIT];

  return (
    <div className="page" style={{ '--page-w': '1040px' }}>
      <div className="page-head">
        <div>
          <h1>状态</h1>
          <p>
            版本 {status.version} · 已运行 {uptime}
          </p>
        </div>
        <div className="page-actions">
          <button type="button" className="btn btn-secondary" onClick={() => toast('状态已刷新')}>
            <Icon name="refresh" />
            刷新
          </button>
        </div>
      </div>

      <div className="adm-stack">
        {warnings.length > 0 && (
          <div className="adm-stack" style={{ gap: 8 }}>
            {warnings.map((w) => (
              <div key={w.code} className="alert warning">
                <Icon name="alert" />
                <div className="grow">
                  <strong>{w.title}</strong>
                  {w.detail && <div>{w.detail}</div>}
                </div>
              </div>
            ))}
          </div>
        )}

        <div className="card adm-metrics">
          <AdmMetric label="图片" value={active.length.toLocaleString()} />
          <AdmMetric label="占用空间" value={fmtBytes(totalBytes)} />
          <AdmMetric label="回收站" value={trashed.length.toLocaleString()} />
          <AdmMetric label="含位置信息" value={gpsCount.toLocaleString()} />
          <AdmMetric label="数据库" value={fmtBytes(db.bytes)} sub={`WAL ${fmtBytes(db.walBytes)}`} />
          <AdmMetric label="磁盘剩余" value={fmtBytes(disk.freeBytes)} sub={`低于 ${Math.round(disk.watermarkBytes / ADM_GB)} GB 时拒绝上传`}>
            <div className="progress" role="progressbar" aria-label="磁盘已用" aria-valuemin="0" aria-valuemax="100" aria-valuenow={usedShare}>
              <span style={{ width: `${usedShare}%` }} />
            </div>
            <span className="s">
              共 {fmtBytes(disk.totalBytes)}，已用 {usedShare}%
            </span>
          </AdmMetric>
        </div>

        <AdmCard title="维护">
          <AdmRunRow
            label="快速检查"
            hint="确认每张图片的文件存在且大小一致"
            idleLabel="开始检查"
            busyLabel="检查中…"
            running={running === 'quick'}
            disabled={!!running}
            onRun={() => runCheck('quick', '1.2', '快速检查')}
            result={results.quick}
          />
          <AdmRunRow
            label="完整检查"
            hint="重新计算每个文件的哈希，并查找没有记录的文件"
            idleLabel="开始检查"
            busyLabel="检查中…"
            running={running === 'full'}
            disabled={!!running}
            onRun={() => runCheck('full', '4.8', '完整检查')}
            result={results.full}
          />
          <AdmRunRow
            label="创建快照"
            hint="把数据库和图片复制到快照目录，用于备份"
            idleLabel="创建快照"
            busyLabel="创建中…"
            running={running === 'snapshot'}
            disabled={!!running}
            onRun={runSnapshot}
            result={results.snapshot}
          />
        </AdmCard>

        <section className="card">
          <div className="card-head">
            <h2>运行信息</h2>
          </div>
          <div className="card-body">
            <dl className="props" style={{ gridTemplateColumns: '96px minmax(0, 1fr)' }}>
              <dt>版本</dt>
              <dd>{status.version}</dd>
              <dt>提交</dt>
              <dd className="mono">{status.commit}</dd>
              <dt>Go</dt>
              <dd>{status.goVersion}</dd>
              <dt>SQLite</dt>
              <dd>{status.sqliteVersion}</dd>
              <dt>数据目录</dt>
              <dd className="mono" style={{ overflowWrap: 'anywhere' }}>
                {status.dataDir}
              </dd>
            </dl>
          </div>
        </section>

        <section className="card" style={{ overflow: 'hidden' }}>
          <div className="card-head">
            <h2>操作记录</h2>
          </div>
          {mobile ? (
            <div className="adm-audit-list">
              {audit.map((a, i) => (
                <div key={i} className="adm-audit-item">
                  <span className="mono subtle">{fmtDate(a.at)}</span>
                  <span>
                    {a.actor} · {a.action}
                  </span>
                  <span className="truncate subtle">{a.target}</span>
                </div>
              ))}
            </div>
          ) : (
            <table className="table adm-table">
              <thead>
                <tr>
                  <th>时间</th>
                  <th>操作者</th>
                  <th>操作</th>
                  <th>对象</th>
                </tr>
              </thead>
              <tbody>
                {audit.map((a, i) => (
                  <tr key={i}>
                    <td className="mono subtle" style={{ whiteSpace: 'nowrap' }}>
                      {fmtDate(a.at)}
                    </td>
                    <td>{a.actor}</td>
                    <td style={{ whiteSpace: 'nowrap' }}>{a.action}</td>
                    <td>
                      <div className="truncate" style={{ maxWidth: 360 }}>
                        {a.target}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </div>
    </div>
  );
}

Object.assign(window, { TokensPage, SettingsPage, StatusPage });
