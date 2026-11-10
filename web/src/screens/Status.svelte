<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from '../components/Icon.svelte';
  import Select from '../components/Select.svelte';
  import { api, errorMessage } from '../lib/api';
  import { formatAbsolute, formatBytes, formatInt, formatRelative } from '../lib/format';
  import { refreshHealth } from '../lib/health.svelte';
  import { phone } from '../lib/media.svelte';
  import { toast } from '../lib/toast.svelte';
  import type { AuditEntry, CheckIssue, CheckKind, Status } from '../lib/types';

  let status = $state<Status | null>(null);
  let phase = $state<'loading' | 'ready' | 'error'>('loading');
  let loadError = $state('');

  /** Returns false when the request failed (the previous status, if any, stays on screen). */
  async function loadStatus(): Promise<boolean> {
    try {
      status = await api.status();
      phase = 'ready';
      return true;
    } catch (e) {
      if (!status) {
        loadError = errorMessage(e);
        phase = 'error';
      }
      return false;
    }
  }

  const HEADLINE: Record<string, string> = {
    'public-url-unset': '未配置公开地址',
    'insecure-public-url': '公开地址未使用 HTTPS',
    'untrusted-forwarded': '收到未受信任代理的转发头',
    'low-disk': '磁盘空间不足',
    'snapshot-stale': '近期没有快照',
    'check-failed': '一致性检查发现问题',
    'non-loopback-listen': '服务监听在非回环地址',
    'wal-large': '数据库日志文件过大',
  };

  const REMEDY: Record<string, string> = {
    'public-url-unset': '设置 PICBERTH_PUBLIC_URL，否则链接可能无法从外部访问。',
    'insecure-public-url': '在反向代理上启用 HTTPS，并把公开地址改为 https。',
    'untrusted-forwarded':
      '收到来自不受信任来源的 X-Forwarded-For。若前面有反向代理，请把它的地址加入受信任代理（PICBERTH_TRUSTED_PROXIES）。',
    'low-disk': '新的上传会被拒绝。清理空间或扩容。',
    'snapshot-stale': '在下方创建快照。',
    'check-failed': '运行完整检查并查看下方结果。',
    'non-loopback-listen': '确认前面有反向代理或防火墙。',
    'wal-large': '创建快照或重启服务通常可以让它缩小。',
  };

  const ACTION: Record<string, string> = {
    setup: '初始化',
    'setup.failure': '初始化失败',
    'login.success': '登录',
    'login.failure': '登录失败',
    logout: '退出',
    'admin.create': '创建管理员',
    'admin.reset-password': '重置密码',
    'token.create': '创建令牌',
    'token.revoke': '吊销令牌',
    'settings.update': '修改设置',
    'image.trash': '移到回收站',
    'image.restore': '恢复',
    'image.purge': '永久删除',
    'image.upload': '上传',
    'image.move': '移动',
    'image.update': '修改图片',
    'image.rename': '重命名',
    'folder.create': '创建文件夹',
    'folder.rename': '重命名文件夹',
    'folder.move': '移动文件夹',
    'folder.delete': '删除文件夹',
    snapshot: '创建快照',
    'check.quick': '快速检查',
    'check.full': '完整检查',
  };

  /** user:<name> -> name, token:<id>:<name> -> name（#id）, system / cli -> 系统 / 命令行. */
  function actorLabel(actor: string): string {
    if (actor.startsWith('user:')) return actor.slice(5);
    if (actor.startsWith('token:')) {
      const [id, ...rest] = actor.slice(6).split(':');
      const name = rest.join(':');
      return name ? `${name}（#${id}）` : `令牌 #${id}`;
    }
    if (actor === 'system') return '系统';
    if (actor === 'cli') return '命令行';
    if (actor === 'anonymous') return '匿名';
    return actor;
  }

  const KIND: Record<CheckKind, string> = {
    'missing-file': '文件缺失',
    'size-mismatch': '大小不符',
    'hash-mismatch': '哈希不符',
    'orphan-file': '孤立文件',
    'pending-move': '未完成的移动',
    'stale-temp': '过期临时文件',
  };

  // ---- maintenance -----------------------------------------------------------------------------
  interface Result {
    tone: string;
    text: string;
    at: string | null;
    path?: string;
    issues?: CheckIssue[];
  }
  type Key = 'quick' | 'full' | 'snapshot';

  let running = $state<Key | null>(null);
  let local = $state<Partial<Record<Key, Result>>>({});

  function remembered(key: Key): Result | null {
    if (!status) return null;
    if (key === 'snapshot') {
      const s = status.lastSnapshot;
      return s ? { tone: 'idle', text: '', at: s.at, path: s.path } : null;
    }
    const c = status.lastCheck;
    if (!c || c.mode !== key) return null;
    return { tone: c.ok ? 'success' : 'danger', text: c.ok ? '正常' : `发现 ${c.issues} 个问题`, at: c.at };
  }
  const result = (key: Key): Result | null => local[key] ?? remembered(key);

  async function runCheck(mode: 'quick' | 'full') {
    if (running) return;
    running = mode;
    try {
      const r = await api.runCheck(mode);
      local[mode] = {
        tone: r.ok ? 'success' : 'danger',
        text: r.ok ? '正常' : `发现 ${r.issues.length} 个问题`,
        at: r.startedAt,
        issues: r.issues,
      };
      toast(`${mode === 'quick' ? '快速检查' : '完整检查'}完成`);
    } catch (e) {
      local[mode] = { tone: 'danger', text: '失败', at: null, path: errorMessage(e) };
    } finally {
      running = null;
      void loadStatus();
      void loadAudit(false, true);
      void refreshHealth();
    }
  }

  async function createSnapshot() {
    if (running) return;
    running = 'snapshot';
    try {
      const s = await api.createSnapshot();
      local.snapshot = { tone: 'success', text: '已创建', at: s.createdAt, path: s.path };
      toast('快照已创建');
    } catch (e) {
      local.snapshot = { tone: 'danger', text: '失败', at: null, path: errorMessage(e) };
    } finally {
      running = null;
      void loadStatus();
      void loadAudit(false, true);
      void refreshHealth();
    }
  }

  // ---- audit -----------------------------------------------------------------------------------
  const PAGE = 10;
  const KIND_OPTIONS = [
    { value: '', label: '全部操作' },
    { value: 'image', label: '图片' },
    { value: 'folder', label: '文件夹' },
    { value: 'token', label: '令牌' },
    { value: 'settings', label: '设置' },
    { value: 'system', label: '系统' },
    { value: 'session', label: '登录' },
  ];

  let audit = $state<AuditEntry[]>([]);
  let auditCursor = $state<string | null>(null);
  let auditPhase = $state<'loading' | 'ready' | 'error'>('loading');
  let auditMore = $state(false);
  let auditError = $state('');
  let kind = $state('');
  let auditRetention = $state<number | null>(null);

  /** `quiet` reloads the first page without swapping the list for a loading line. */
  async function loadAudit(more = false, quiet = false) {
    if (more) auditMore = true;
    else if (!quiet || auditPhase !== 'ready') auditPhase = 'loading';
    try {
      const page = await api.audit(PAGE, more ? auditCursor : null, kind);
      audit = more ? [...audit, ...page.items] : page.items;
      auditCursor = page.nextCursor;
      auditPhase = 'ready';
    } catch (e) {
      auditError = errorMessage(e);
      if (!more && !quiet) auditPhase = 'error';
    } finally {
      auditMore = false;
    }
  }

  function setKind(value: string) {
    if (value === kind) return;
    kind = value;
    audit = [];
    auditCursor = null;
    void loadAudit(false);
  }

  let refreshing = $state(false);
  async function refresh() {
    if (refreshing) return;
    refreshing = true;
    try {
      const [ok] = await Promise.all([loadStatus(), loadAudit(false, true), refreshHealth()]);
      toast(ok ? '状态已刷新' : '刷新失败，显示的是上一次读取的状态');
    } finally {
      refreshing = false;
    }
  }

  onMount(() => {
    void loadStatus();
    void loadAudit();
    api
      .getSettings()
      .then((s) => (auditRetention = s.auditRetentionDays))
      .catch(() => {});
  });

  // ---- display ---------------------------------------------------------------------------------
  const used = $derived(status ? Math.max(0, status.disk.totalBytes - status.disk.freeBytes) : 0);
  /** 图片 / 数据库 / 其他 / 可用: 其他 is what is used beyond the first two, never below 0. */
  const parts = $derived.by(() => {
    if (!status) return [];
    const other = Math.max(0, used - status.images.bytes - status.database.bytes);
    return [
      { k: 'images', label: '图片', v: status.images.bytes },
      { k: 'db', label: '数据库', v: status.database.bytes },
      { k: 'other', label: '其他', v: other },
      { k: 'free', label: '可用', v: status.disk.freeBytes },
    ];
  });
  const uptime = $derived.by(() => {
    if (!status) return '';
    const days = Math.floor(status.uptimeSeconds / 86400);
    return days > 0 ? `${days} 天` : `${Math.max(1, Math.floor(status.uptimeSeconds / 3600))} 小时`;
  });
  const warnings = $derived(status ? status.warnings : []);
  const MAINT: { k: Key; name: string; btn: string; busy: string }[] = [
    { k: 'quick', name: '快速检查', btn: '检查', busy: '检查中…' },
    { k: 'full', name: '完整检查', btn: '检查', busy: '检查中…' },
    { k: 'snapshot', name: '快照', btn: '创建', busy: '创建中…' },
  ];
</script>

<div class="panel-main">
  <div class="head">
    <div class="head-title">
      <h1>状态</h1>
    </div>
    <button type="button" class="btn btn-ghost" disabled={refreshing || phase === 'loading'} onclick={() => void refresh()}>
      <Icon name="refresh" />
      刷新
    </button>
  </div>
  <div class="scroll">
    {#if phase === 'loading' || phase === 'error' || !status}
      <div class="page page-wide">
        {#if phase === 'error'}
          <div class="state" role="alert">
            <span class="t">状态加载失败</span>
            <span class="d">{loadError}</span>
            <button type="button" class="btn btn-secondary" onclick={() => void loadStatus()}>重试</button>
          </div>
        {:else}
          <div class="skeleton" style="height: 64px;" aria-busy="true"></div>
        {/if}
      </div>
    {:else}
      <div class="page page-wide">
        {#if warnings.length > 0}
          <section aria-label="警告">
            {#each warnings as w (w.code)}
              <div class="warn-row" role="alert">
                <span class="dot dot-warning"></span>
                <span class="t">{HEADLINE[w.code] ?? w.message}</span>
                {#if REMEDY[w.code]}<span class="d">{REMEDY[w.code]}</span>{/if}
              </div>
            {/each}
          </section>
        {/if}

        <div class="figures">
          <div class="figure">
            <span class="v">{formatInt(status.images.active)}</span>
            <span class="k">图片</span>
          </div>
          <div class="figure">
            <span class="v">{formatBytes(status.images.bytes)}</span>
            <span class="k">占用空间</span>
          </div>
          <div class="figure">
            <span class="v">{formatInt(status.images.trashed)}</span>
            <span class="k">回收站</span>
          </div>
          <div class="figure">
            <span class="v">{formatInt(status.images.broken)}</span>
            <span class="k">损坏</span>
          </div>
          <div class="figure">
            <span class="v">{formatBytes(status.database.bytes)}</span>
            <span class="k">数据库</span>
          </div>
        </div>

        <section class="section">
          <div class="disk-head">
            <h2>磁盘</h2>
            <span class="muted num">已用 {formatBytes(used)} / {formatBytes(status.disk.totalBytes)}</span>
          </div>
          <div class="stack-bar" role="img" aria-label={parts.map((p) => `${p.label} ${formatBytes(p.v)}`).join('，')}>
            {#each parts as p (p.k)}
              <i class="sw-{p.k}" style:width={p.k === 'free' || status.disk.totalBytes <= 0 ? undefined : `${(p.v / status.disk.totalBytes) * 100}%`}></i>
            {/each}
          </div>
          <dl class="legend">
            {#each parts as p (p.k)}
              <div>
                <dt><i class="sw-{p.k}"></i>{p.label}</dt>
                <dd>{formatBytes(p.v)}</dd>
              </div>
            {/each}
          </dl>
        </section>

        <section class="section">
          <h2>维护</h2>
          <div class="list">
            {#each MAINT as m (m.k)}
              {@const r = result(m.k)}
              <div class="list-row maint-row">
                <span class="medium">{m.name}</span>
                <span class="c-res badge">
                  {#if r?.tone}<span class="dot dot-{r.tone}"></span>{/if}
                  {#if r?.text}<span title={r.path}>{r.text}</span>{/if}
                  {#if r?.at}<span class="num" title={formatAbsolute(r.at)}>{formatRelative(r.at)}</span>{/if}
                </span>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  disabled={running !== null}
                  onclick={() => void (m.k === 'snapshot' ? createSnapshot() : runCheck(m.k as 'quick' | 'full'))}
                >
                  {running === m.k ? m.busy : m.btn}
                </button>
              </div>
              {#if running !== m.k && r?.issues && r.issues.length > 0}
                <div class="list" role="table" aria-label="检查结果">
                  <div class="list-row list-head audit-row" role="row">
                    <span>类型</span>
                    <span>图片</span>
                    <span>路径</span>
                    <span>状态</span>
                  </div>
                  {#each r.issues as issue, i (i)}
                    <div class="list-row audit-row" role="row">
                      <span>{KIND[issue.kind] ?? issue.kind}</span>
                      <span class="mono ellipsis" title={issue.imageId ?? undefined}>{issue.imageId ?? '—'}</span>
                      <span class="mono ellipsis" title={issue.path ?? undefined}>{issue.path ?? '—'}</span>
                      <span>
                        {#if issue.repaired}<span class="text-success">已修复</span>{:else}<span class="text-warning">未修复</span>{/if}
                      </span>
                    </div>
                  {/each}
                </div>
              {/if}
            {/each}
          </div>
        </section>

        <section class="section">
          <div class="section-head">
            <div class="head-title">
              <h2>操作记录</h2>
              {#if auditRetention !== null}<span class="s muted">保留 {auditRetention} 天</span>{/if}
            </div>
            <Select label="操作类型" value={kind} options={KIND_OPTIONS} width={128} onchange={setKind} />
          </div>
          {#if auditPhase === 'loading'}
            <div class="list" aria-busy="true">
              {#each [0, 1, 2] as i (i)}
                <div class="list-row audit-row">
                  <div class="skeleton" style="height: 10px; width: 60%;"></div>
                </div>
              {/each}
            </div>
          {:else if auditPhase === 'error'}
            <div class="state" role="alert">
              <span class="t">记录加载失败</span>
              <span class="d">{auditError}</span>
              <button type="button" class="btn btn-secondary" onclick={() => void loadAudit()}>重试</button>
            </div>
          {:else if audit.length === 0}
            <div class="state" style="padding: 32px 0;">
              <span class="t">没有匹配的记录</span>
            </div>
          {:else}
            <div class="list" role="table" aria-label="操作记录">
              {#if !phone.matches}
                <div class="list-row list-head audit-row" role="row">
                  <span>时间</span>
                  <span>操作者</span>
                  <span>操作</span>
                  <span>对象</span>
                </div>
              {/if}
              {#each audit as a (a.id)}
                <div class="list-row audit-row" role="row">
                  <span class="c-at muted num" title={formatAbsolute(a.at)}>{formatRelative(a.at)}</span>
                  <span class="c-actor ellipsis" title={a.actor}>{actorLabel(a.actor)}</span>
                  <span class="c-action medium">{ACTION[a.action] ?? a.action}</span>
                  <span class="c-target ellipsis muted" title={a.target ?? undefined}>{a.target ?? ''}</span>
                </div>
              {/each}
            </div>
            {#if auditCursor}
              <div>
                <button
                  type="button"
                  class="btn btn-ghost"
                  style="margin-left: -12px;"
                  disabled={auditMore}
                  onclick={() => void loadAudit(true)}
                >
                  {auditMore ? '加载中…' : '加载更多'}
                </button>
              </div>
            {/if}
          {/if}
        </section>

        <section class="section">
          <h2>运行信息</h2>
          <dl class="dl">
            <dt>版本</dt>
            <dd class="num row" style="gap: 12px;">
              <span>{status.version}</span>
              {#if status.commit}<span class="mono muted">{status.commit}</span>{/if}
            </dd>
            <dt>运行环境</dt>
            <dd class="num row" style="gap: 12px;">
              <span>{status.goVersion}</span>
              <span>SQLite {status.sqliteVersion}</span>
            </dd>
            <dt>已运行</dt>
            <dd>{uptime}</dd>
            <dt>数据目录</dt>
            <dd class="mono" style="font-size: var(--fs-m);">{status.dataDir}</dd>
          </dl>
        </section>
      </div>
    {/if}
  </div>
</div>
