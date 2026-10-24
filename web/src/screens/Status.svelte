<script lang="ts">
  import { onMount } from 'svelte';
  import AdmRow from '../components/AdmRow.svelte';
  import AdmSection from '../components/AdmSection.svelte';
  import Icon from '../components/Icon.svelte';
  import PanelHeader from '../components/PanelHeader.svelte';
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
    'public-url-unset': '设置公开地址（PICBERTH_PUBLIC_URL），否则生成的链接可能无法在外部访问。',
    'insecure-public-url': '公开地址使用 http。请在反向代理上启用 HTTPS，并把公开地址改为 https。',
    'untrusted-forwarded':
      '收到来自不受信任来源的 X-Forwarded-For。若前面有反向代理，请把它的地址加入受信任代理（PICBERTH_TRUSTED_PROXIES）。',
    'low-disk': '剩余磁盘空间低于水位线，新的上传会被拒绝。请清理空间或扩容。',
    'snapshot-stale': '最近没有新的快照。请在下方创建快照，并考虑定期执行。',
    'check-failed': '最近一次检查发现了问题。请运行完整检查并查看下方结果。',
    'non-loopback-listen': '服务监听在非本地回环地址。请确认前面有反向代理或防火墙保护。',
    'wal-large': '数据库的 WAL 文件过大，可能有长时间未结束的读取。创建快照或重启服务通常可以让它缩小。',
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
    'image.visibility': '修改可见性',
    'image.update': '修改图片',
    'image.rename': '重命名',
    'image.tags': '修改标签',
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
    badge?: string;
    tone?: string;
    text?: string;
    extra?: string;
    title?: string;
    path?: string;
    subtle?: boolean;
    issues?: CheckIssue[];
  }
  type Key = 'quick' | 'full' | 'snapshot';

  let running = $state<Key | null>(null);
  let local = $state<Partial<Record<Key, Result>>>({});

  const secs = (ms: number) => `${(ms / 1000).toFixed(1)} 秒`;

  function remembered(key: Key): Result | null {
    if (!status) return null;
    if (key === 'snapshot') {
      // A stale snapshot shows its warning on this row instead of in a separate alert.
      const s = status.lastSnapshot;
      const stale = status.warnings.find((w) => w.code === 'snapshot-stale');
      if (stale) return { badge: HEADLINE[stale.code] ?? stale.message, tone: 'badge-warning', text: s ? formatRelative(s.at) : undefined, title: s?.path, subtle: true };
      return s ? { subtle: true, text: formatRelative(s.at), title: s.path } : null;
    }
    const c = status.lastCheck;
    if (!c || c.mode !== key) return null;
    return {
      badge: c.ok ? '正常' : `发现 ${c.issues} 个问题`,
      tone: c.ok ? 'badge-success' : 'badge-danger',
      text: formatRelative(c.at),
      title: formatAbsolute(c.at),
      subtle: true,
    };
  }
  const result = (key: Key): Result | null => local[key] ?? remembered(key);

  async function runCheck(mode: 'quick' | 'full') {
    if (running) return;
    running = mode;
    try {
      const r = await api.runCheck(mode);
      local[mode] = {
        badge: r.ok ? '正常' : `发现 ${r.issues.length} 个问题`,
        tone: r.ok ? 'badge-success' : 'badge-danger',
        text: `${formatInt(r.checked)} 张图片`,
        extra: secs(r.durationMs),
        issues: r.issues,
      };
      toast(`${mode === 'quick' ? '快速检查' : '完整检查'}完成`);
    } catch (e) {
      local[mode] = { badge: '失败', tone: 'badge-danger', text: errorMessage(e) };
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
      local.snapshot = { badge: '已创建', tone: 'badge-success', path: s.path, extra: formatBytes(s.bytes) };
      toast('快照已创建');
    } catch (e) {
      local.snapshot = { badge: '失败', tone: 'badge-danger', text: errorMessage(e) };
    } finally {
      running = null;
      void loadStatus();
      void loadAudit(false, true);
      void refreshHealth();
    }
  }

  // ---- audit -----------------------------------------------------------------------------------
  let audit = $state<AuditEntry[]>([]);
  let auditCursor = $state<string | null>(null);
  let auditPhase = $state<'loading' | 'ready' | 'error'>('loading');
  let auditMore = $state(false);
  let auditError = $state('');

  /** `quiet` reloads the first page without swapping the list for a loading line. */
  async function loadAudit(more = false, quiet = false) {
    if (more) auditMore = true;
    else if (!quiet || auditPhase !== 'ready') auditPhase = 'loading';
    try {
      const page = await api.audit(20, more ? auditCursor : null);
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
  });

  // ---- display ---------------------------------------------------------------------------------
  const usedShare = $derived(
    status && status.disk.totalBytes > 0 ? Math.round(((status.disk.totalBytes - status.disk.freeBytes) / status.disk.totalBytes) * 100) : 0,
  );
  const uptime = $derived.by(() => {
    if (!status) return '';
    const days = Math.floor(status.uptimeSeconds / 86400);
    return days > 0 ? `${days} 天` : `${Math.max(1, Math.floor(status.uptimeSeconds / 3600))} 小时`;
  });
</script>

{#snippet runRow(key: Key, label: string, idle: string, busy: string, run: () => void)}
  {@const r = result(key)}
  <div>
    <div class="adm-row">
      <span class="label">{label}</span>
      <div class="adm-run">
        {#if running === key}
          <div class="progress indeterminate adm-run-bar" role="progressbar" aria-label={label}><span></span></div>
        {:else if r}
          <div class="adm-result" class:subtle={r.subtle}>
            {#if r.badge}<span class="badge {r.tone ?? 'badge-success'}">{r.badge}</span>{/if}
            {#if r.text}<span title={r.title}>{r.text}</span>{/if}
            {#if r.path}<span class="mono">{r.path}</span>{/if}
            {#if r.extra}<span class="subtle">{r.extra}</span>{/if}
          </div>
        {/if}
        <button type="button" class="btn btn-secondary" disabled={running !== null} onclick={run}>
          {running === key ? busy : idle}
        </button>
      </div>
    </div>
    {#if running !== key && r?.issues && r.issues.length > 0}
      <div class="adm-issues" role="table" aria-label="检查结果">
        <div class="adm-grid head" role="row">
          <span role="columnheader">类型</span>
          <span role="columnheader">图片</span>
          <span role="columnheader">路径</span>
          <span role="columnheader">状态</span>
        </div>
        {#each r.issues as issue, i (i)}
          <div class="adm-grid" role="row">
            <span role="cell" class="nowrap">{KIND[issue.kind] ?? issue.kind}</span>
            <span role="cell" class="mono truncate" title={issue.imageId ?? undefined}>{issue.imageId ?? '—'}</span>
            <span role="cell" class="mono truncate" title={issue.path ?? undefined}>{issue.path ?? '—'}</span>
            <span role="cell">
              {#if issue.repaired}<span class="badge badge-success">已修复</span>{:else}<span class="badge badge-warning">未修复</span>{/if}
            </span>
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/snippet}

<div class="page">
  <PanelHeader title="状态">
    <button type="button" class="btn btn-ghost" disabled={refreshing || phase === 'loading'} onclick={() => void refresh()}>
      <Icon name="refresh" />
      刷新
    </button>
  </PanelHeader>
  <div class="page-body">
    <div class="adm-col">
      {#if phase === 'loading'}
        <p class="subtle">加载中…</p>
      {:else if phase === 'error' || !status}
        <div class="alert danger" role="alert">
          <Icon name="alert" />
          <div class="grow">加载失败：{loadError}</div>
          <button type="button" class="btn btn-secondary btn-sm" onclick={() => void loadStatus()}>重试</button>
        </div>
      {:else}
        {#each status.warnings.filter((w) => w.code !== 'snapshot-stale') as w (w.code)}
          <div class="alert warning">
            <Icon name="alert" />
            <div class="grow">
              <strong>{HEADLINE[w.code] ?? w.message}</strong>
              {#if REMEDY[w.code]}<div>{REMEDY[w.code]}</div>{/if}
            </div>
          </div>
        {/each}

        <div class="stats">
          <div class="stat">
            <span class="k">图片</span>
            <span class="v">{formatInt(status.images.active)}</span>
            {#if status.images.broken > 0}<span class="s">{formatInt(status.images.broken)} 张异常</span>{/if}
          </div>
          <div class="stat">
            <span class="k">占用空间</span>
            <span class="v">{formatBytes(status.images.bytes)}</span>
          </div>
          <div class="stat">
            <span class="k">回收站</span>
            <span class="v">{formatInt(status.images.trashed)}</span>
          </div>
          <div class="stat">
            <span class="k">数据库</span>
            <span class="v">{formatBytes(status.database.bytes)}</span>
          </div>
          <div class="stat">
            <span class="k">磁盘剩余</span>
            <span class="v">{formatBytes(status.disk.freeBytes)}</span>
            <div class="progress" role="progressbar" aria-label="磁盘已用" aria-valuemin={0} aria-valuemax={100} aria-valuenow={usedShare}>
              <span style:width="{usedShare}%"></span>
            </div>
          </div>
        </div>

        <AdmSection title="维护">
          {@render runRow('quick', '快速检查', '检查', '检查中…', () => void runCheck('quick'))}
          {@render runRow('full', '完整检查', '检查', '检查中…', () => void runCheck('full'))}
          {@render runRow('snapshot', '快照', '创建', '创建中…', () => void createSnapshot())}
        </AdmSection>

        <AdmSection title="运行信息" compact>
          <AdmRow label="版本">
            <div class="adm-value adm-pair">
              <span>{status.version}</span>
              {#if status.commit}<span class="mono subtle">{status.commit}</span>{/if}
            </div>
          </AdmRow>
          <AdmRow label="已运行"><div class="adm-value">{uptime}</div></AdmRow>
          <AdmRow label="Go"><div class="adm-value">{status.goVersion}</div></AdmRow>
          <AdmRow label="SQLite"><div class="adm-value">{status.sqliteVersion}</div></AdmRow>
          <AdmRow label="数据目录"><div class="adm-value mono">{status.dataDir}</div></AdmRow>
        </AdmSection>
      {/if}

      <section class="adm-sec">
        <h2>操作记录</h2>
        {#if auditPhase === 'loading'}
          <p class="subtle">加载中…</p>
        {:else if auditPhase === 'error'}
          <div class="alert danger" role="alert">
            <Icon name="alert" />
            <div class="grow">加载失败：{auditError}</div>
            <button type="button" class="btn btn-secondary btn-sm" onclick={() => void loadAudit()}>重试</button>
          </div>
        {:else if audit.length === 0}
          <p class="subtle">暂无记录</p>
        {:else}
          {#if phone.matches}
            <div class="adm-audit-list">
              {#each audit as a (a.id)}
                <div class="adm-audit-item">
                  <span class="mono subtle" title={formatRelative(a.at)}>{formatAbsolute(a.at)}</span>
                  <span class="adm-pair">
                    <span>{ACTION[a.action] ?? a.action}</span>
                    <span class="subtle">{actorLabel(a.actor)}</span>
                  </span>
                  {#if a.target}<span class="truncate subtle">{a.target}</span>{/if}
                  {#if a.detail || a.ip}
                    <span class="adm-pair subtle">
                      {#if a.detail}<span>{a.detail}</span>{/if}
                      {#if a.ip}<span class="mono">{a.ip}</span>{/if}
                    </span>
                  {/if}
                </div>
              {/each}
            </div>
          {:else}
            <div class="adm-audit" role="table" aria-label="操作记录">
              <div class="adm-grid head" role="row">
                <span role="columnheader">时间</span>
                <span role="columnheader">操作者</span>
                <span role="columnheader">操作</span>
                <span role="columnheader">对象</span>
              </div>
              {#each audit as a (a.id)}
                <div class="adm-grid" role="row">
                  <span role="cell" class="mono subtle nowrap" title={formatRelative(a.at)}>{formatAbsolute(a.at)}</span>
                  <span role="cell" class="truncate" title={a.actor}>{actorLabel(a.actor)}</span>
                  <span role="cell" class="nowrap">{ACTION[a.action] ?? a.action}</span>
                  <span role="cell">
                    <span class="truncate" title={a.target ?? undefined}>{a.target ?? ''}</span>
                    {#if a.detail || a.ip}
                      <span class="adm-pair sub">
                        {#if a.detail}<span class="truncate">{a.detail}</span>{/if}
                        {#if a.ip}<span class="mono">{a.ip}</span>{/if}
                      </span>
                    {/if}
                  </span>
                </div>
              {/each}
            </div>
          {/if}
          {#if auditCursor}
            <div class="adm-more">
              <button type="button" class="btn btn-secondary" disabled={auditMore} onclick={() => void loadAudit(true)}>
                {auditMore ? '加载中…' : '加载更多'}
              </button>
            </div>
          {/if}
        {/if}
      </section>
    </div>
  </div>
</div>