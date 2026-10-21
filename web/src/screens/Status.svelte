<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import Button from '../components/Button.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import PageHeader from '../components/PageHeader.svelte';
  import { api, errorMessage } from '../lib/api';
  import { formatAbsolute, formatBytes, formatDuration, formatInt, formatRelative } from '../lib/format';
  import { refreshHealth } from '../lib/health.svelte';
  import { toast } from '../lib/toast.svelte';
  import type { AuditEntry, CheckKind, CheckReport, Snapshot, Status } from '../lib/types';

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

  const REMEDY: Record<string, string> = {
    'public-url-unset': '设置公开地址（PICBERTH_PUBLIC_URL），否则生成的链接可能无法在外部访问。',
    'insecure-public-url': '公开地址使用 http。请在反向代理上启用 HTTPS，并把公开地址改为 https。',
    'untrusted-forwarded': '收到来自不受信任来源的 X-Forwarded-For。若前面有反向代理，请把它的地址加入受信任代理（PICBERTH_TRUSTED_PROXIES）。',
    'low-disk': '剩余磁盘空间低于水位线，新的上传会被拒绝。请清理空间或扩容。',
    'snapshot-stale': '最近没有新的快照。请在下方创建快照，并考虑定期执行。',
    'check-failed': '最近一次检查发现了问题。请运行完整检查并查看下方结果。',
    'non-loopback-listen': '服务监听在非本地回环地址。请确认前面有反向代理或防火墙保护。',
    'wal-large': '数据库的 WAL 文件过大，可能有长时间未结束的读取。创建快照或重启服务通常可以让它缩小。',
  };

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
    snapshot: '创建快照',
    'check.quick': '快速检查',
    'check.full': '完整检查',
  };

  /** user:<name> -> name, token:<id>:<name> -> 令牌 name（#id）, system / cli -> 系统 / 命令行. */
  function actorLabel(actor: string): string {
    if (actor.startsWith('user:')) return actor.slice(5);
    if (actor.startsWith('token:')) {
      const [id, ...rest] = actor.slice(6).split(':');
      const name = rest.join(':');
      return name ? `令牌 ${name}（#${id}）` : `令牌 #${id}`;
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

  // ---- checks ----------------------------------------------------------------------------------
  let running = $state<'quick' | 'full' | null>(null);
  let elapsed = $state(0);
  let report = $state<CheckReport | null>(null);
  let checkError = $state('');
  /** The mode of the most recent run, so an error is shown under the right row. */
  let checkMode = $state<'quick' | 'full' | null>(null);
  let timer: ReturnType<typeof setInterval> | undefined;

  async function runCheck(mode: 'quick' | 'full') {
    if (running || snapshotting) return;
    running = mode;
    checkMode = mode;
    elapsed = 0;
    checkError = '';
    const t0 = Date.now();
    timer = setInterval(() => (elapsed = Math.floor((Date.now() - t0) / 1000)), 500);
    try {
      report = await api.runCheck(mode);
    } catch (e) {
      report = null;
      checkError = errorMessage(e);
    } finally {
      clearInterval(timer);
      running = null;
      void loadStatus();
      void refreshHealth();
    }
  }

  let snapshotting = $state(false);
  let snapshot = $state<Snapshot | null>(null);
  let snapshotError = $state('');
  async function createSnapshot() {
    if (running || snapshotting) return;
    snapshotting = true;
    snapshotError = '';
    const t0 = Date.now();
    elapsed = 0;
    timer = setInterval(() => (elapsed = Math.floor((Date.now() - t0) / 1000)), 500);
    try {
      snapshot = await api.createSnapshot();
    } catch (e) {
      snapshotError = errorMessage(e);
    } finally {
      clearInterval(timer);
      snapshotting = false;
      void loadStatus();
      void refreshHealth();
    }
  }
  onDestroy(() => clearInterval(timer));

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
      if (!ok && status) toast('刷新失败，显示的是上一次读取的状态');
    } finally {
      refreshing = false;
    }
  }

  onMount(() => {
    void loadStatus();
    void loadAudit();
  });

  // ---- derived display -------------------------------------------------------------------------
  function parts(bytes: number): { n: string; u: string } {
    const s = formatBytes(bytes);
    const i = s.lastIndexOf(' ');
    return i < 0 ? { n: s, u: '' } : { n: s.slice(0, i), u: s.slice(i + 1) };
  }

  function dur(ms: number): string {
    return ms < 1000 ? `${Math.round(ms)} 毫秒` : `${(ms / 1000).toFixed(1)} 秒`;
  }

  const verdict = $derived.by((): { lamp: 'ok' | 'warn' | 'danger'; text: string } | null => {
    if (!status) return null;
    if (status.images.broken > 0 || (status.lastCheck && !status.lastCheck.ok)) return { lamp: 'danger', text: '发现问题' };
    if (status.warnings.length > 0) return { lamp: 'warn', text: `需要注意 · ${status.warnings.length} 项` };
    return { lamp: 'ok', text: '一切正常' };
  });

  interface Figure {
    key: string;
    label: string;
    n: string;
    u: string;
    sub: string;
    mono?: boolean;
    bad?: boolean;
  }
  const figures = $derived.by((): Figure[] => {
    if (!status) return [];
    const bytes = parts(status.images.bytes);
    const db = parts(status.database.bytes);
    const free = parts(status.disk.freeBytes);
    return [
      {
        key: 'active',
        label: '底片',
        n: formatInt(status.images.active),
        u: '张',
        sub: `${formatInt(status.images.broken)} 张异常`,
        bad: status.images.broken > 0,
      },
      { key: 'bytes', label: '占用空间', n: bytes.n, u: bytes.u, sub: '图片文件合计' },
      { key: 'trashed', label: '回收站', n: formatInt(status.images.trashed), u: '张', sub: '等待自动清除' },
      { key: 'gps', label: '含位置', n: formatInt(status.images.withGps), u: '张', sub: '公开前请留意' },
      { key: 'db', label: '数据库', n: db.n, u: db.u, sub: `WAL ${formatBytes(status.database.walBytes)}`, mono: true },
      { key: 'free', label: '磁盘剩余', n: free.n, u: free.u, sub: `共 ${formatBytes(status.disk.totalBytes)}`, mono: true },
    ];
  });

  const diskUsedPct = $derived(
    status && status.disk.totalBytes > 0
      ? Math.min(100, Math.max(0, ((status.disk.totalBytes - status.disk.freeBytes) / status.disk.totalBytes) * 100))
      : 0,
  );
  const hasWatermark = $derived(!!status && status.disk.totalBytes > 0 && status.disk.watermarkBytes > 0);
  const watermarkPct = $derived(
    status && status.disk.totalBytes > 0
      ? Math.min(100, Math.max(0, ((status.disk.totalBytes - status.disk.watermarkBytes) / status.disk.totalBytes) * 100))
      : 100,
  );
  const TICKS = Array.from({ length: 11 }, (_, i) => i * 10);
</script>

<div class="page">
  <PageHeader title="状态" meta={status ? status.version : ''} mono>
    {#snippet actions()}
      <Button disabled={refreshing || phase === 'loading'} onclick={() => void refresh()}>{refreshing ? '刷新中…' : '刷新'}</Button>
    {/snippet}
  </PageHeader>

  {#if phase === 'loading'}
    <p class="muted">加载中…</p>
  {:else if phase === 'error' || !status}
    <EmptyState message={`加载失败：${loadError}`}>
      <Button onclick={() => void loadStatus()}>重试</Button>
    </EmptyState>
  {:else}
    {#snippet checkResult(mode: 'quick' | 'full')}
      {#if checkError && checkMode === mode}
        <p class="rline"><span class="lamp danger"></span><span>检查失败：{checkError}</span></p>
      {:else if report && report.mode === mode}
        <p class="rline">
          <span class="lamp" class:danger={!report.ok}></span>
          <span>
            {#if report.ok}{mode === 'full' ? '完整检查' : '快速检查'}通过{:else}发现 {report.issues.length} 个问题{/if}
            <span class="sum num">· 检查 {formatInt(report.checked)} 项 · {dur(report.durationMs)}</span>
          </span>
        </p>
        {#if report.issues.length > 0}
          <table class="issues mono">
            <thead>
              <tr><th>类型</th><th>图片</th><th>路径</th><th>状态</th></tr>
            </thead>
            <tbody>
              {#each report.issues as issue, i (i)}
                <tr>
                  <td class="kind">{KIND[issue.kind] ?? issue.kind}</td>
                  <td>{issue.imageId ?? '—'}</td>
                  <td class="path">{issue.path ?? '—'}</td>
                  <td>{#if issue.repaired}<span class="stamp ok">已修复</span>{:else}<span class="stamp warn">未修复</span>{/if}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      {:else if status?.lastCheck && status.lastCheck.mode === mode}
        <p class="rline last">
          <span class="lamp" class:danger={!status.lastCheck.ok}></span>
          <span>
            上次检查 <span title={formatAbsolute(status.lastCheck.at)}>{formatRelative(status.lastCheck.at)}</span>
            · {status.lastCheck.ok ? '正常' : `${status.lastCheck.issues} 个问题`}
          </span>
        </p>
      {:else if mode === 'quick' && !status?.lastCheck && !report}
        <p class="rline last"><span class="lamp off"></span><span>尚未运行过检查</span></p>
      {/if}
    {/snippet}

    {#if verdict}
      <p class="verdict display" role="status">
        <span class="lamp vl {verdict.lamp}"></span>
        {verdict.text}
      </p>
    {/if}

    {#if status.warnings.length > 0}
      <ul class="warns" aria-label="警告">
        {#each status.warnings as w (w.code)}
          <li>
            <span class="lamp warn"></span>
            <div>
              <p class="wmsg">{HEADLINE[w.code] ?? w.message}</p>
              {#if REMEDY[w.code]}<p class="wfix">{REMEDY[w.code]}</p>{/if}
            </div>
          </li>
        {/each}
      </ul>
    {/if}

    <section class="figures" aria-label="概览">
      {#each figures as f (f.key)}
        <div class="fig">
          <p class="lbl">{f.label}</p>
          <p class="val">
            <span class="display n">{f.n}</span>{#if f.u}<span class="u">{f.u}</span>{/if}
          </p>
          <p class="sub" class:bad={f.bad} class:mono={f.mono}>{f.sub}</p>
        </div>
      {/each}
    </section>

    <section class="disk" aria-label="磁盘用量">
      <div class="disk-head">
        <p class="lbl">磁盘用量</p>
        <p class="sub num">已用 {diskUsedPct.toFixed(0)}%</p>
      </div>
      <div
        class="scale"
        class:marked={hasWatermark}
        role="img"
        aria-label="已用 {diskUsedPct.toFixed(0)}%，警戒线 {formatBytes(status.disk.watermarkBytes)}"
        title={hasWatermark ? `警戒线 ${formatBytes(status.disk.watermarkBytes)}` : undefined}
      >
        {#if hasWatermark}
          <span class="wm-label edge" class:end={watermarkPct > 90} class:start={watermarkPct < 10} style:left="{watermarkPct}%">警戒线</span>
        {/if}
        <div class="bar">
          <span class="used" style:width="{diskUsedPct}%"></span>
          {#if hasWatermark}<span class="wm" style:left="{watermarkPct}%"></span>{/if}
        </div>
        <div class="ticks" aria-hidden="true">
          {#each TICKS as t (t)}
            <span class="tk" class:major={t % 50 === 0} class:last={t === 100} style:left="{t}%"></span>
          {/each}
        </div>
        <div class="tl edge" aria-hidden="true">
          <span>0</span><span>50</span><span>100</span>
        </div>
      </div>
    </section>

    <section class="block">
      <h2 class="section-title">维护</h2>

      <div class="mgroup">
        <div class="mrow">
          <div class="mtext">
            <p class="mname">快速检查</p>
            <p class="mdesc">确认每条记录的文件存在且大小一致，完成未完成的移动，清理过期的临时文件。</p>
          </div>
          <Button aria-label="运行快速检查" disabled={running !== null || snapshotting} onclick={() => runCheck('quick')}>
            {running === 'quick' ? `检查中 ${elapsed} 秒` : '运行'}
          </Button>
        </div>
        <div class="result" aria-live="polite">{@render checkResult('quick')}</div>
      </div>

      <div class="mgroup">
        <div class="mrow">
          <div class="mtext">
            <p class="mname">完整检查</p>
            <p class="mdesc">在快速检查之外重新计算每个文件的哈希，并遍历目录查找孤立文件。孤立文件会被移到隔离区，不会被删除。</p>
          </div>
          <Button aria-label="运行完整检查" disabled={running !== null || snapshotting} onclick={() => runCheck('full')}>
            {running === 'full' ? `检查中 ${elapsed} 秒` : '运行'}
          </Button>
        </div>
        <div class="result" aria-live="polite">{@render checkResult('full')}</div>
      </div>

      <div class="mgroup">
        <div class="mrow">
          <div class="mtext">
            <p class="mname">创建快照</p>
            <p class="mdesc">把数据库和全部图片文件复制到快照目录，用于备份与恢复。</p>
          </div>
          <Button disabled={running !== null || snapshotting} onclick={createSnapshot}>
            {snapshotting ? `创建中 ${elapsed} 秒` : '创建'}
          </Button>
        </div>
        <div class="result" aria-live="polite">
          {#if snapshotError}
            <p class="rline"><span class="lamp danger"></span><span>快照失败：{snapshotError}</span></p>
          {:else if snapshot}
            <p class="rline">
              <span class="lamp"></span>
              <span>快照已创建 <span class="sum num">· {formatInt(snapshot.images)} 张 · {formatBytes(snapshot.bytes)}</span></span>
            </p>
            <p class="path mono pline">{snapshot.path}</p>
          {:else if status.lastSnapshot}
            <p class="rline last">
              <span class="lamp"></span>
              <span>上次快照 <span title={formatAbsolute(status.lastSnapshot.at)}>{formatRelative(status.lastSnapshot.at)}</span></span>
            </p>
            <p class="path mono pline">{status.lastSnapshot.path}</p>
          {:else}
            <p class="rline last"><span class="lamp off"></span><span>尚未创建过快照</span></p>
          {/if}
        </div>
      </div>
    </section>

    <section class="block">
      <h2 class="section-title">运行信息</h2>
      <dl class="kv">
        <dt>版本</dt>
        <dd class="mono">{status.version}</dd>
        <dt>提交</dt>
        <dd class="mono">{status.commit || '—'}</dd>
        <dt>Go</dt>
        <dd class="mono">{status.goVersion}</dd>
        <dt>SQLite</dt>
        <dd class="mono">{status.sqliteVersion}</dd>
        <dt>运行时间</dt>
        <dd class="mono" title="自 {formatAbsolute(status.startedAt)}">{formatDuration(status.uptimeSeconds)}</dd>
        <dt>数据目录</dt>
        <dd class="mono">{status.dataDir}</dd>
      </dl>
    </section>
  {/if}

  <section class="block">
    <h2 class="section-title">操作记录</h2>
    {#if auditPhase === 'loading'}
      <p class="muted pad">加载中…</p>
    {:else if auditPhase === 'error'}
      <EmptyState message={`加载失败：${auditError}`}>
        <Button onclick={() => void loadAudit()}>重试</Button>
      </EmptyState>
    {:else if audit.length === 0}
      <p class="muted pad">暂无记录</p>
    {:else}
      <ol class="log">
        {#each audit as a (a.id)}
          <li>
            <span class="t mono" title={formatRelative(a.at)}>{formatAbsolute(a.at)}</span>
            <span class="who" title={a.actor}>{actorLabel(a.actor)}</span>
            <span class="act" title={a.action}>{ACTION[a.action] ?? a.action}</span>
            {#if a.target || a.detail || a.ip}
              <span class="what">
                {#if a.target}<span class="tgt mono" title={a.target}>{a.target}</span>{/if}
                {#if a.detail || a.ip}
                  <span class="det">{[a.detail, a.ip].filter(Boolean).join(' · ')}</span>
                {/if}
              </span>
            {/if}
          </li>
        {/each}
      </ol>
      {#if auditCursor}
        <div class="more">
          <Button disabled={auditMore} onclick={() => void loadAudit(true)}>{auditMore ? '加载中…' : '加载更多'}</Button>
        </div>
      {/if}
    {/if}
  </section>
</div>

<style>
  .num {
    font-variant-numeric: tabular-nums;
  }

  /* ---- verdict + warnings ---- */
  .verdict {
    display: flex;
    align-items: center;
    gap: var(--s3);
    margin-bottom: var(--s4);
    font-size: var(--t-28);
  }
  .vl {
    width: 12px;
    height: 12px;
    margin-top: 2px;
  }
  .warns {
    margin-bottom: var(--s6);
    border-top: 1px solid var(--line);
  }
  .warns li {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    gap: var(--s3);
    padding: 14px 0;
    border-bottom: 1px solid var(--line);
  }
  .warns .lamp {
    margin-top: 7px;
  }
  .wmsg {
    font-weight: 600;
  }
  .wfix {
    margin-top: 2px;
    color: var(--ink-2);
    font-size: var(--t-13);
  }

  /* ---- figures: a ledger of hero numbers ---- */
  .figures {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1px;
    margin: var(--s5) 0 var(--s6);
    border-block: 1px solid var(--line);
    background: var(--line);
  }
  .fig {
    min-width: 0;
    padding: 18px 20px 20px;
    background: var(--bg);
  }
  .fig:nth-child(3n + 1) {
    padding-left: 0;
  }
  .lbl {
    color: var(--ink-3);
    font-size: var(--t-12);
  }
  .val {
    display: flex;
    align-items: baseline;
    gap: 6px;
    margin-top: 6px;
    line-height: 1;
  }
  .n {
    font-size: var(--t-44);
    font-variant-numeric: tabular-nums;
    overflow-wrap: normal;
  }
  .u {
    color: var(--ink-3);
    font-size: var(--t-14);
    font-weight: 450;
  }
  .sub {
    margin-top: 8px;
    color: var(--ink-3);
    font-size: 12.5px;
    font-variant-numeric: tabular-nums;
  }
  .sub.mono {
    font-size: 12px;
  }
  .bad {
    color: var(--danger);
  }

  /* ---- exposure scale ---- */
  .disk {
    margin-bottom: var(--s7);
  }
  .disk-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--s3);
  }
  .disk-head .sub {
    margin: 0;
  }
  .scale {
    position: relative;
    margin-top: var(--s2);
    padding-top: 6px;
  }
  .scale.marked {
    padding-top: 22px;
  }
  .bar {
    position: relative;
    height: 10px;
    background: var(--surface-2);
  }
  .used {
    position: absolute;
    inset: 0 auto 0 0;
    background: var(--ink);
  }
  .wm {
    position: absolute;
    top: -5px;
    bottom: -5px;
    width: 2px;
    margin-left: -1px;
    background: var(--mark);
  }
  .wm-label {
    position: absolute;
    top: 0;
    transform: translateX(-50%);
    color: var(--mark);
    font-size: 10.5px;
    font-weight: 600;
  }
  .wm-label.end {
    transform: translateX(calc(-100% + 1px));
  }
  .wm-label.start {
    transform: translateX(-1px);
  }
  .ticks {
    position: relative;
    height: 8px;
    margin-top: 4px;
  }
  .tk {
    position: absolute;
    top: 0;
    width: 1px;
    height: 4px;
    background: var(--line-strong);
  }
  .tk.major {
    height: 8px;
    background: var(--ink-3);
  }
  .tk.last {
    margin-left: -1px;
  }
  .tl {
    position: relative;
    height: 11px;
    margin-top: 5px;
    color: var(--ink-3);
  }
  .tl span {
    position: absolute;
    top: 0;
  }
  .tl span:nth-child(1) {
    left: 0;
  }
  .tl span:nth-child(2) {
    left: 50%;
    transform: translateX(-50%);
  }
  .tl span:nth-child(3) {
    right: 0;
  }

  /* ---- sections ---- */
  .block {
    margin-bottom: var(--s7);
  }
  .mrow {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--s5);
    padding: 18px 0;
  }
  .mgroup {
    border-bottom: 1px solid var(--line);
  }
  .mname {
    font-weight: 600;
  }
  .mdesc {
    max-width: 62ch;
    margin-top: 3px;
    color: var(--ink-3);
    font-size: var(--t-13);
  }
  .result {
    font-size: var(--t-13);
  }
  .result:not(:empty) {
    padding: 0 0 18px;
  }
  .result:empty {
    display: none;
  }
  .rline {
    display: flex;
    align-items: baseline;
    gap: var(--s2);
    padding: var(--s2) 0 0;
  }
  .rline:first-child {
    padding-top: 0;
  }
  .rline .lamp {
    position: relative;
    top: -1px;
    align-self: center;
  }
  .rline.last {
    color: var(--ink-3);
  }
  .sum {
    color: var(--ink-3);
  }
  .pline {
    margin-top: 4px;
    padding-left: calc(8px + var(--s2));
    color: var(--ink-3);
    font-size: 12px;
  }
  .path {
    overflow-wrap: anywhere;
  }

  .issues {
    margin-top: var(--s3);
    font-size: 12px;
  }
  .issues th,
  .issues td {
    padding: 7px var(--s3) 7px 0;
    border-bottom: 1px solid var(--line);
    vertical-align: top;
  }
  .issues thead th {
    border-bottom-color: var(--line-strong);
    color: var(--ink-3);
    font-family: var(--font-ui);
    font-size: var(--t-12);
    font-weight: 550;
  }
  .issues tbody tr:last-child td {
    border-bottom: 0;
  }
  .kind {
    font-family: var(--font-ui);
    white-space: nowrap;
  }
  .issues .stamp {
    font-family: var(--font-ui);
    height: 18px;
    font-size: 11px;
  }

  .kv {
    display: grid;
    grid-template-columns: 140px minmax(0, 1fr);
    column-gap: var(--s5);
    font-size: var(--t-13);
  }
  .kv dt,
  .kv dd {
    padding: 10px 0;
    border-bottom: 1px solid var(--line);
  }
  .kv dt {
    color: var(--ink-3);
  }

  .pad {
    padding: var(--s3) 0;
  }
  .log li {
    display: grid;
    grid-template-columns: 138px 150px 112px minmax(0, 1fr);
    column-gap: var(--s4);
    align-items: baseline;
    padding: 10px 0;
    border-bottom: 1px solid var(--line);
    font-size: var(--t-13);
  }
  .t {
    color: var(--ink-3);
    font-size: 12.5px;
    white-space: nowrap;
  }
  .who,
  .tgt,
  .det {
    display: block;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .who {
    color: var(--ink-2);
  }
  .act {
    font-weight: 600;
  }
  .what {
    min-width: 0;
  }
  .tgt {
    font-size: 12.5px;
  }
  .det {
    color: var(--ink-3);
    font-size: 12px;
  }
  .more {
    padding-top: var(--s4);
  }

  @media (max-width: 720px) {
    .figures {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .fig:nth-child(3n + 1) {
      padding-left: 20px;
    }
    .fig:nth-child(2n + 1) {
      padding-left: 0;
    }
    .fig {
      padding-right: 12px;
    }
    .mrow {
      align-items: flex-start;
      flex-direction: column;
      gap: var(--s3);
    }
    .kv {
      grid-template-columns: 88px minmax(0, 1fr);
      column-gap: var(--s3);
    }
    .log li {
      grid-template-columns: minmax(0, 1fr) auto;
      grid-template-areas:
        'act t'
        'who who'
        'what what';
      row-gap: 2px;
    }
    .act {
      grid-area: act;
    }
    .t {
      grid-area: t;
    }
    .who {
      grid-area: who;
    }
    .what {
      grid-area: what;
    }
    .issues thead {
      display: none;
    }
    .issues,
    .issues tbody {
      display: block;
    }
    .issues tr {
      display: grid;
      gap: 2px;
      padding: var(--s2) 0;
      border-bottom: 1px solid var(--line);
    }
    .issues td {
      display: block;
      padding: 0;
      border: 0;
    }
  }
  @media (max-width: 560px) {
    .n {
      font-size: 34px;
    }
    .verdict {
      font-size: 24px;
    }
  }
</style>
