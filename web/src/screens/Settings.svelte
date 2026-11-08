<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import Icon from '../components/Icon.svelte';
  import Modal from '../components/Modal.svelte';
  import Seg from '../components/Seg.svelte';
  import Switch from '../components/Switch.svelte';
  import { api, errorMessage } from '../lib/api';
  import { FORMAT_NAMES } from '../lib/format';
  import { refreshHealth } from '../lib/health.svelte';
  import { phone } from '../lib/media.svelte';
  import { canLeave, addGuard } from '../lib/router.svelte';
  import { logout, refreshSession, session } from '../lib/session.svelte';
  import { setTheme, theme, THEME_OPTIONS } from '../lib/theme.svelte';
  import { bottomBar, toast } from '../lib/toast.svelte';
  import type { ImageFormat, Settings } from '../lib/types';

  const ALL_FORMATS: ImageFormat[] = ['jpeg', 'png', 'gif', 'webp', 'avif'];
  const MB = 1024 * 1024;

  // Numbers are edited as text so partial input never fights the user.
  interface Form {
    maxMB: string;
    maxMP: string;
    formats: Record<ImageFormat, boolean>;
    dedupe: boolean;
    retention: string;
    perMinute: string;
    idle: string;
    absolute: string;
  }
  type NumberKey = 'maxMB' | 'maxMP' | 'retention' | 'perMinute' | 'idle' | 'absolute';

  const trimNum = (n: number) => String(Math.round(n * 1e6) / 1e6);
  function toForm(s: Settings): Form {
    return {
      maxMB: trimNum(s.maxUploadBytes / MB),
      maxMP: trimNum(s.maxPixels / 1e6),
      formats: Object.fromEntries(ALL_FORMATS.map((f) => [f, s.allowedFormats.includes(f)])) as Record<ImageFormat, boolean>,
      dedupe: s.dedupe,
      retention: String(s.trashRetentionDays),
      perMinute: String(s.uploadsPerMinute),
      idle: String(s.sessionIdleMinutes),
      absolute: String(s.sessionAbsoluteHours),
    };
  }

  let baseline = $state<Settings | null>(null);
  let phase = $state<'loading' | 'ready' | 'error'>('loading');
  let loadError = $state('');
  let form = $state<Form>({
    maxMB: '',
    maxMP: '',
    formats: { jpeg: false, png: false, gif: false, webp: false, avif: false },
    dedupe: false,
    retention: '',
    perMinute: '',
    idle: '',
    absolute: '',
  });

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

  // What the form would send, in API units. Every image is public, so no visibility field exists.
  const values = $derived({
    maxUploadBytes: Math.round(num(form.maxMB) * MB),
    maxPixels: Math.round(num(form.maxMP) * 1e6),
    allowedFormats: ALL_FORMATS.filter((f) => form.formats[f]),
    dedupe: form.dedupe,
    trashRetentionDays: num(form.retention),
    uploadsPerMinute: num(form.perMinute),
    sessionIdleMinutes: num(form.idle),
    sessionAbsoluteHours: num(form.absolute),
  });

  const range = (lo: number, hi: number) => `请输入 ${lo} 到 ${hi} 之间的数字`;
  const intIn = (v: number, lo: number, hi: number) => (Number.isInteger(v) && v >= lo && v <= hi ? '' : range(lo, hi));

  const errors = $derived.by((): Record<string, string> => {
    if (!baseline) return {};
    const hard = baseline.hardMaxUploadBytes;
    const mb = num(form.maxMB);
    const mp = num(form.maxMP);
    return {
      maxMB: Number.isNaN(mb) || values.maxUploadBytes < 1024 || values.maxUploadBytes > hard ? range(1, Math.floor(hard / MB)) : '',
      maxMP: Number.isNaN(mp) || mp < 1 || mp > 200 ? range(1, 200) : '',
      formats: values.allowedFormats.length === 0 ? '至少保留一种格式' : '',
      retention: intIn(values.trashRetentionDays, 0, 365),
      perMinute: intIn(values.uploadsPerMinute, 1, 10000),
      idle: intIn(values.sessionIdleMinutes, 5, 10080),
      absolute: intIn(values.sessionAbsoluteHours, 1, 8760),
    };
  });
  const invalid = $derived(Object.values(errors).some(Boolean));

  function same(a: unknown, b: unknown): boolean {
    return Array.isArray(a) && Array.isArray(b) ? [...a].sort().join() === [...b].sort().join() : a === b;
  }
  const changed = $derived.by(() => {
    if (!baseline) return [] as (keyof typeof values)[];
    const base = baseline;
    return (Object.keys(values) as (keyof typeof values)[]).filter((k) => !same(values[k], base[k]));
  });
  const dirty = $derived(changed.length > 0);

  let saving = $state(false);
  let saveError = $state('');

  async function save() {
    if (!dirty || invalid || saving) return;
    saving = true;
    saveError = '';
    // PATCH carries only the fields that changed.
    const patch: Record<string, unknown> = {};
    for (const k of changed) patch[k] = values[k];
    try {
      const s = await api.patchSettings(patch as Parameters<typeof api.patchSettings>[0]);
      baseline = s;
      form = toForm(s);
      toast('设置已保存');
      void refreshSession(); // upload limits shown in the upload dialog may have changed
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
    leaveOpen = false;
    if (ok) discard(); // 放弃更改: a later guard check (e.g. sign-out) sees no unsaved work
    leaveResolve?.(ok);
    leaveResolve = null;
  }
  // On phones the bar is fixed above the tab bar, so toasts have to clear it.
  $effect(() => {
    bottomBar.active = phone.matches && dirty;
    return () => (bottomBar.active = false);
  });

  const removeGuard = addGuard(async () => (dirty ? await askLeave() : true));
  onDestroy(removeGuard);

  $effect(() => {
    if (!dirty) return;
    const warn = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = '';
    };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  });

  // ---- account (phone only: the phone layout has no sidebar) -----------------------------------
  // Same sign-out flow as the Shell account menu; the settings leave guard covers unsaved changes.
  async function signOut() {
    if (!(await canLeave('/login'))) return;
    try {
      await logout();
      toast('已退出登录');
    } catch {
      toast('退出失败，请重试');
    }
  }
</script>

{#snippet numberField(key: NumberKey, id: string, unit: string)}
  <div class="field-wrap has-suffix w-num">
    <input
      {id}
      class="input num"
      inputmode={key === 'maxMB' || key === 'maxMP' ? 'decimal' : 'numeric'}
      aria-invalid={errors[key] ? 'true' : undefined}
      bind:value={form[key]}
    />
    <span class="suffix">{unit}</span>
  </div>
  {#if errors[key]}<span class="field-error">{errors[key]}</span>{/if}
{/snippet}

<div class="panel-main">
  <div class="head">
    <div class="head-title">
      <h1>设置</h1>
    </div>
  </div>
  <div class="scroll">
    <div class="page">
      {#if phase === 'loading'}
        <div class="page" aria-busy="true">
          <div class="skeleton" style="height: 76px; border-radius: 8px;"></div>
        </div>
      {:else if phase === 'error' || !baseline}
        <div class="state" role="alert" style="padding: 64px 0;">
          <span class="t">设置加载失败</span>
          <span class="d">{loadError}</span>
          <button type="button" class="btn btn-secondary" onclick={() => void load()}>重试</button>
        </div>
      {:else}
        <section class="section">
          <h2>上传</h2>
          <div class="form-rows">
            <div class="form-row">
              <label class="k" for="s-max">单文件上限</label>
              <div class="ctl">
                {@render numberField('maxMB', 's-max', 'MB')}
              </div>
            </div>
            <div class="form-row">
              <label class="k" for="s-px">像素上限</label>
              <div class="ctl">
                {@render numberField('maxMP', 's-px', 'MP')}
              </div>
            </div>
            <div class="form-row stack">
              <span class="k">格式</span>
              <div class="ctl">
                {#each ALL_FORMATS as f (f)}
                  <button
                    type="button"
                    class="chip"
                    aria-pressed={form.formats[f]}
                    onclick={() => (form.formats[f] = !form.formats[f])}
                  >
                    {#if form.formats[f]}<Icon name="check" size="s" />{/if}
                    {FORMAT_NAMES[f]}
                  </button>
                {/each}
                {#if errors.formats}<span class="field-error">{errors.formats}</span>{/if}
              </div>
            </div>
            <div class="form-row">
              <label class="k" for="s-rate">频率上限</label>
              <div class="ctl">
                {@render numberField('perMinute', 's-rate', '次/分')}
              </div>
            </div>
            <div class="form-row">
              <span class="k">重复检测</span>
              <div class="ctl">
                <Switch checked={form.dedupe} label="重复检测" onchange={(v) => (form.dedupe = v)} />
              </div>
            </div>
          </div>
        </section>

        <section class="section">
          <h2>回收站</h2>
          <div class="form-rows">
            <div class="form-row">
              <label class="k" for="s-trash">保留时间</label>
              <div class="ctl">
                {@render numberField('retention', 's-trash', '天')}
              </div>
            </div>
          </div>
        </section>

        <section class="section">
          <h2>会话</h2>
          <div class="form-rows">
            <div class="form-row">
              <label class="k" for="s-idle">空闲超时</label>
              <div class="ctl">
                {@render numberField('idle', 's-idle', '分钟')}
              </div>
            </div>
            <div class="form-row">
              <label class="k" for="s-abs">最长有效</label>
              <div class="ctl">
                {@render numberField('absolute', 's-abs', '小时')}
              </div>
            </div>
          </div>
        </section>

        <section class="section">
          <h2>地址</h2>
          <div class="form-rows">
            <div class="form-row">
              <span class="k">公开地址</span>
              <div class="ctl">
                <span class="mono ellipsis" style="font-size: var(--fs-m);">{baseline.publicUrl || '—'}</span>
              </div>
            </div>
            <div class="form-row">
              <span class="k">图片地址</span>
              <div class="ctl">
                <span class="mono ellipsis" style="font-size: var(--fs-m);">{baseline.imageUrl || '—'}</span>
              </div>
            </div>
          </div>
        </section>

        {#if phone.matches}
          <section class="section">
            <h2>账户</h2>
            <div class="form-rows">
              <div class="form-row">
                <span class="k">用户名</span>
                <div class="ctl">
                  <span>{session.user?.name ?? ''}</span>
                </div>
              </div>
              <div class="form-row">
                <span class="k">主题</span>
                <div class="ctl">
                  <Seg label="主题" value={theme.choice} onchange={setTheme} options={THEME_OPTIONS.map((t) => ({ key: t.key, label: t.label }))} />
                </div>
              </div>
            </div>
            <div>
              <button type="button" class="btn btn-ghost" style="margin-left: -12px" onclick={() => void signOut()}>
                <Icon name="logout" />
                退出
              </button>
            </div>
          </section>
        {/if}
      {/if}
    </div>
  </div>
  {#if dirty && phase === 'ready'}
    <div class="save-bar" role="region" aria-label="未保存的更改">
      <span class="t">有未保存的更改</span>
      <button type="button" class="btn btn-ghost" onclick={discard}>放弃</button>
      <button type="button" class="btn btn-primary" disabled={invalid || saving} onclick={() => void save()}>保存</button>
    </div>
  {/if}
</div>

{#if leaveOpen}
  <Modal title="离开设置？" onclose={() => answer(false)}>
    <p>未保存的更改会丢失。</p>
    {#snippet footer()}
      <button type="button" class="btn btn-secondary" data-autofocus onclick={() => answer(false)}>留下</button>
      <button type="button" class="btn btn-danger" onclick={() => answer(true)}>放弃更改</button>
    {/snippet}
  </Modal>
{/if}
