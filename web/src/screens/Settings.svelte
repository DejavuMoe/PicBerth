<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import Button from '../components/Button.svelte';
  import ConfirmDialog from '../components/ConfirmDialog.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import Grease from '../components/Grease.svelte';
  import PageHeader from '../components/PageHeader.svelte';
  import Segmented from '../components/Segmented.svelte';
  import SettingRow from '../components/SettingRow.svelte';
  import Switch from '../components/Switch.svelte';
  import TextField from '../components/TextField.svelte';
  import { api, errorMessage } from '../lib/api';
  import { FORMAT_NAMES, formatLimit } from '../lib/format';
  import { refreshHealth } from '../lib/health.svelte';
  import { addGuard } from '../lib/router.svelte';
  import { refreshSession } from '../lib/session.svelte';
  import { bottomBar, toast } from '../lib/toast.svelte';
  import type { ImageFormat, Settings, Visibility, WritableSettings } from '../lib/types';

  const ALL_FORMATS: ImageFormat[] = ['jpeg', 'png', 'gif', 'webp', 'avif'];
  const MB = 1024 * 1024;

  const SECTIONS = [
    { id: 'sec-upload', label: '上传' },
    { id: 'sec-trash', label: '回收站' },
    { id: 'sec-rate', label: '频率限制' },
    { id: 'sec-session', label: '会话' },
    { id: 'sec-address', label: '地址' },
  ];

  let baseline = $state<Settings | null>(null);
  let phase = $state<'loading' | 'ready' | 'error'>('loading');
  let loadError = $state('');

  // Numbers are edited as text so partial input never fights the user.
  interface Form {
    maxMB: string;
    maxMP: string;
    formats: Record<ImageFormat, boolean>;
    defaultVisibility: Visibility;
    dedupe: boolean;
    retention: string;
    perMinute: string;
    idle: string;
    absolute: string;
  }

  const trimNum = (n: number) => String(Math.round(n * 1e6) / 1e6);
  function toForm(s: Settings): Form {
    return {
      maxMB: trimNum(s.maxUploadBytes / MB),
      maxMP: trimNum(s.maxPixels / 1e6),
      formats: Object.fromEntries(ALL_FORMATS.map((f) => [f, s.allowedFormats.includes(f)])) as Record<ImageFormat, boolean>,
      defaultVisibility: s.defaultVisibility,
      dedupe: s.dedupe,
      retention: String(s.trashRetentionDays),
      perMinute: String(s.uploadsPerMinute),
      idle: String(s.sessionIdleMinutes),
      absolute: String(s.sessionAbsoluteHours),
    };
  }
  let form = $state<Form | null>(null);

  async function load() {
    phase = 'loading';
    try {
      const s = await api.getSettings();
      baseline = s;
      form = toForm(s);
      phase = 'ready';
    } catch (e) {
      loadError = errorMessage(e);
      phase = 'error';
    }
  }
  onMount(() => void load());

  const num = (v: string) => (/^\s*\d+(\.\d+)?\s*$/.test(v) ? Number(v) : NaN);

  const VIS_OPTIONS = [
    { value: 'public', label: '公开' },
    { value: 'private', label: '私有' },
  ] satisfies { value: Visibility; label: string }[];

  // What the form would send, in API units.
  const values = $derived.by((): WritableSettings | null => {
    if (!form) return null;
    return {
      maxUploadBytes: Math.round(num(form.maxMB) * MB),
      maxPixels: Math.round(num(form.maxMP) * 1e6),
      allowedFormats: ALL_FORMATS.filter((f) => form!.formats[f]),
      dedupe: form.dedupe,
      defaultVisibility: form.defaultVisibility,
      trashRetentionDays: num(form.retention),
      uploadsPerMinute: num(form.perMinute),
      sessionIdleMinutes: num(form.idle),
      sessionAbsoluteHours: num(form.absolute),
    };
  });

  function intIn(v: number, lo: number, hi: number, unit = ''): string | null {
    return Number.isInteger(v) && v >= lo && v <= hi ? null : `需要 ${lo} – ${hi}${unit} 之间的整数`;
  }

  const errors = $derived.by(() => {
    if (!values || !baseline || !form) return {} as Record<string, string | null>;
    const hard = baseline.hardMaxUploadBytes;
    const mb = num(form.maxMB);
    const mp = num(form.maxMP);
    return {
      maxMB:
        Number.isNaN(mb) || values.maxUploadBytes < 1024 || values.maxUploadBytes > hard
          ? `需要在 1 KB 到 ${formatLimit(hard)} 之间`
          : null,
      maxMP: Number.isNaN(mp) || mp < 1 || mp > 200 ? '需要在 1 到 200 之间' : null,
      formats: values.allowedFormats.length === 0 ? '至少允许一种格式' : null,
      retention: intIn(values.trashRetentionDays, 0, 365),
      perMinute: intIn(values.uploadsPerMinute, 1, 10000),
      idle: intIn(values.sessionIdleMinutes, 5, 10080),
      absolute: intIn(values.sessionAbsoluteHours, 1, 8760),
    } as Record<string, string | null>;
  });
  const invalid = $derived(Object.values(errors).some((e) => !!e));

  function same(a: unknown, b: unknown): boolean {
    return Array.isArray(a) && Array.isArray(b) ? [...a].sort().join() === [...b].sort().join() : a === b;
  }
  const changed = $derived.by(() => {
    if (!values || !baseline) return [] as (keyof WritableSettings)[];
    const keys = Object.keys(values) as (keyof WritableSettings)[];
    return keys.filter((k) => !same(values[k], baseline![k]));
  });
  const dirty = $derived(changed.length > 0);

  let saving = $state(false);
  let saveError = $state('');

  async function save() {
    if (!values || !dirty || invalid || saving) return;
    saving = true;
    saveError = '';
    // PATCH carries only the fields that changed.
    const patch: Record<string, unknown> = {};
    for (const k of changed) patch[k] = values[k];
    try {
      const s = await api.patchSettings(patch as Partial<WritableSettings>);
      baseline = s;
      form = toForm(s);
      toast('设置已保存');
      void refreshSession(); // upload limits shown on the home screen may have changed
      void refreshHealth();
    } catch (e) {
      saveError = errorMessage(e);
    } finally {
      saving = false;
    }
  }
  function discard() {
    if (baseline) form = toForm(baseline);
    saveError = '';
  }

  // ---- leaving with unsaved changes ------------------------------------------------------------
  let leaveOpen = $state(false);
  let leaveResolve: ((ok: boolean) => void) | null = null;
  function askLeave(): Promise<boolean> {
    leaveOpen = true;
    return new Promise((resolve) => (leaveResolve = resolve));
  }
  function answer(ok: boolean) {
    leaveResolve?.(ok);
    leaveResolve = null;
  }
  const removeGuard = addGuard(async () => (dirty ? await askLeave() : true));
  onDestroy(() => {
    removeGuard();
    bottomBar.active = false;
  });

  $effect(() => {
    if (!dirty) return;
    const warn = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = '';
    };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  });
  $effect(() => {
    bottomBar.active = dirty;
  });

  // ---- section index ---------------------------------------------------------------------------
  let activeSection = $state('sec-upload');

  function jump(e: MouseEvent, id: string) {
    e.preventDefault(); // keep the router out of in-page anchors
    activeSection = id;
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    document.getElementById(id)?.scrollIntoView({ behavior: reduce ? 'auto' : 'smooth', block: 'start' });
  }

  $effect(() => {
    if (phase !== 'ready' || typeof IntersectionObserver === 'undefined') return;
    const seen = new Set<string>();
    const io = new IntersectionObserver(
      (entries) => {
        for (const en of entries) {
          if (en.isIntersecting) seen.add(en.target.id);
          else seen.delete(en.target.id);
        }
        const first = SECTIONS.find((s) => seen.has(s.id));
        if (first) activeSection = first.id;
      },
      { rootMargin: '-88px 0px -60% 0px' },
    );
    for (const s of SECTIONS) {
      const el = document.getElementById(s.id);
      if (el) io.observe(el);
    }
    return () => io.disconnect();
  });
</script>

<div class="page">
  <PageHeader title="设置" meta="修改后立即对新的上传生效" />

  {#if phase === 'loading'}
    <p class="muted">加载中…</p>
  {:else if phase === 'error' || !form || !baseline}
    <EmptyState message={`加载失败：${loadError}`}>
      <Button onclick={() => void load()}>重试</Button>
    </EmptyState>
  {:else}
    <form
      class:dirty
      onsubmit={(e) => {
        e.preventDefault();
        void save();
      }}
      novalidate
    >
      <div class="layout">
        <nav class="index" aria-label="设置分区">
          <ul>
            {#each SECTIONS as s (s.id)}
              <li>
                <a href={`#${s.id}`} class:on={activeSection === s.id} aria-current={activeSection === s.id ? 'location' : undefined} onclick={(e) => jump(e, s.id)}>
                  <span class="lbl">
                    {s.label}
                    {#if activeSection === s.id}<Grease shape="line" seed={s.id} />{/if}
                  </span>
                </a>
              </li>
            {/each}
          </ul>
        </nav>

        <div class="sections">
          <section id="sec-upload" aria-labelledby="h-upload">
            <h2 class="section-title" id="h-upload">上传</h2>
            <SettingRow label="单文件上限" help="超过此大小的文件会被拒绝">
              <div class="narrow">
                <TextField
                  label="单文件上限"
                  hideLabel
                  inputmode="decimal"
                  suffix="MB"
                  hint={`最大 ${formatLimit(baseline.hardMaxUploadBytes)}`}
                  error={errors.maxMB}
                  bind:value={form.maxMB}
                />
              </div>
            </SettingRow>
            <SettingRow label="像素上限" help="单帧宽 × 高的上限，防止超大图片占满内存">
              <div class="narrow">
                <TextField
                  label="像素上限"
                  hideLabel
                  inputmode="decimal"
                  suffix="百万像素"
                  hint="1 – 200"
                  error={errors.maxMP}
                  bind:value={form.maxMP}
                />
              </div>
            </SettingRow>
            <SettingRow label="允许的格式" help="不在列表中的格式会被拒绝">
              <fieldset class="formats">
                <legend class="sr-only">允许的格式</legend>
                {#each ALL_FORMATS as fmt (fmt)}
                  <button
                    type="button"
                    class="fmt mono"
                    class:on={form.formats[fmt]}
                    aria-pressed={form.formats[fmt]}
                    onclick={() => form && (form.formats[fmt] = !form.formats[fmt])}
                  >
                    {FORMAT_NAMES[fmt] ?? fmt}
                  </button>
                {/each}
              </fieldset>
              {#if errors.formats}<p class="err">{errors.formats}</p>{/if}
            </SettingRow>
            <SettingRow label="默认可见性" help="上传页面和未指定时新图片的可见性">
              <Segmented
                label="默认可见性"
                options={VIS_OPTIONS}
                value={form.defaultVisibility}
                onchange={(v) => form && (form.defaultVisibility = v)}
              />
            </SettingRow>
            <SettingRow label="重复检测" help="内容相同的图片只保存一份，上传时返回原链接" labelId="dedupe-label">
              <Switch labelledby="dedupe-label" checked={form.dedupe} onchange={(on) => form && (form.dedupe = on)} />
            </SettingRow>
          </section>

          <section id="sec-trash" aria-labelledby="h-trash">
            <h2 class="section-title" id="h-trash">回收站</h2>
            <SettingRow label="保留天数" help="0 表示立即清除，不进入回收站">
              <div class="narrow">
                <TextField label="保留天数" hideLabel inputmode="numeric" suffix="天" error={errors.retention} bind:value={form.retention} />
              </div>
            </SettingRow>
          </section>

          <section id="sec-rate" aria-labelledby="h-rate">
            <h2 class="section-title" id="h-rate">频率限制</h2>
            <SettingRow label="每个令牌每分钟上传次数" help="浏览器会话按用户计算">
              <div class="narrow">
                <TextField
                  label="每个令牌每分钟上传次数"
                  hideLabel
                  inputmode="numeric"
                  suffix="次"
                  error={errors.perMinute}
                  bind:value={form.perMinute}
                />
              </div>
            </SettingRow>
          </section>

          <section id="sec-session" aria-labelledby="h-session">
            <h2 class="section-title" id="h-session">会话</h2>
            <SettingRow label="空闲超时" help="无操作超过此时间后需要重新登录">
              <div class="narrow">
                <TextField label="空闲超时" hideLabel inputmode="numeric" suffix="分钟" error={errors.idle} bind:value={form.idle} />
              </div>
            </SettingRow>
            <SettingRow label="最长有效" help="无论是否活跃，登录超过此时间后失效">
              <div class="narrow">
                <TextField label="最长有效" hideLabel inputmode="numeric" suffix="小时" error={errors.absolute} bind:value={form.absolute} />
              </div>
            </SettingRow>
          </section>

          <section id="sec-address" aria-labelledby="h-address">
            <h2 class="section-title" id="h-address">地址</h2>
            <SettingRow label="公开地址">
              <p class="ro mono">{baseline.publicUrl || '—'}</p>
            </SettingRow>
            <SettingRow label="图片地址">
              <p class="ro mono">{baseline.imageUrl || '—'}</p>
            </SettingRow>
            <p class="note">在配置文件或环境变量中修改，修改后已发出的链接会失效</p>
          </section>
        </div>
      </div>

      {#if dirty}
        <div class="savebar" role="region" aria-label="未保存的更改">
          <p class="msg" aria-live="polite">
            <span class="count"><span class="n display">{changed.length}</span> 项更改未保存</span>
            {#if saveError}<span class="err">{saveError}</span>{/if}
          </p>
          <div class="btns">
            <Button variant="inverse" onclick={discard} disabled={saving}>放弃</Button>
            <Button type="submit" variant="print" disabled={saving || invalid}>保存</Button>
          </div>
        </div>
      {/if}
    </form>
  {/if}
</div>

<ConfirmDialog
  bind:open={leaveOpen}
  title="离开此页？"
  message="有未保存的更改，离开后将丢失。"
  confirmLabel="离开"
  cancelLabel="留下"
  danger
  onconfirm={() => answer(true)}
  oncancel={() => answer(false)}
/>

<style>
  form.dirty {
    padding-bottom: calc(var(--s7) + var(--s4));
  }

  .layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
  }
  .index {
    display: none;
  }
  .sections {
    min-width: 0;
  }
  section {
    margin-bottom: var(--s7);
    scroll-margin-top: calc(var(--bar-h) + 24px);
  }
  section:last-child {
    margin-bottom: 0;
  }

  @media (min-width: 900px) {
    .layout {
      grid-template-columns: 148px minmax(0, 1fr);
      column-gap: var(--s7);
      align-items: start;
    }
    .index {
      display: block;
      position: sticky;
      top: calc(var(--bar-h) + 24px);
      padding-top: 4px;
    }
  }

  .index ul {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .index a {
    display: inline-block;
    padding: 4px 0;
    color: var(--ink-2);
    font-size: var(--t-14);
    text-decoration: none;
    transition: color var(--dur) var(--ease);
  }
  .index a:hover {
    color: var(--ink);
  }
  .index a.on {
    color: var(--ink);
    font-weight: 650;
  }
  .lbl {
    position: relative;
    display: inline-block;
  }

  .narrow {
    max-width: 240px;
  }

  .formats {
    display: flex;
    flex-wrap: wrap;
    gap: var(--s2);
    min-width: 0;
    padding: 0;
    border: 0;
  }
  .fmt {
    min-width: 58px;
    height: 30px;
    padding: 0 12px;
    border: 1px solid var(--line-strong);
    border-radius: var(--r-chip);
    color: var(--ink-3);
    font-size: var(--t-12);
    font-weight: 560;
    line-height: 1;
    transition:
      background-color var(--dur) var(--ease),
      border-color var(--dur) var(--ease),
      color var(--dur) var(--ease);
  }
  .fmt:hover {
    border-color: var(--ink);
    color: var(--ink);
  }
  .fmt.on {
    background: var(--ink);
    border-color: var(--ink);
    color: var(--bg);
  }

  .err {
    margin-top: var(--s2);
    color: var(--danger);
    font-size: var(--t-13);
  }
  .ro {
    font-size: var(--t-13);
    user-select: all;
  }
  .note {
    margin-top: var(--s3);
    color: var(--ink-3);
    font-size: var(--t-13);
  }

  /* Unsaved changes: the same film slip as the selection bar. */
  .savebar {
    position: fixed;
    left: 50%;
    bottom: calc(16px + var(--tabbar-h));
    z-index: 40;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--s2) var(--s4);
    width: min(640px, calc(100vw - 2 * var(--s3)));
    min-height: 56px;
    padding: 6px 6px 6px 18px;
    border-radius: var(--r-ctl);
    background: var(--film);
    color: var(--print);
    font-size: var(--t-13);
    box-shadow:
      0 0 0 1px var(--film-line),
      var(--shadow);
    animation: bar-in 240ms var(--ease-out);
  }
  .msg {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: var(--s1) var(--s3);
    min-width: 0;
  }
  .count {
    white-space: nowrap;
  }
  .n {
    margin-right: 2px;
    color: var(--edge);
    font-size: 26px;
    font-variant-numeric: tabular-nums;
  }
  .savebar .err {
    margin: 0;
    color: var(--mark);
  }
  .btns {
    display: flex;
    flex: none;
    gap: 2px;
  }
  @keyframes bar-in {
    from {
      opacity: 0;
      transform: translate(-50%, 14px);
    }
  }
</style>
