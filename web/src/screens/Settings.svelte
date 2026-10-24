<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import AdmRow from '../components/AdmRow.svelte';
  import AdmSection from '../components/AdmSection.svelte';
  import Icon from '../components/Icon.svelte';
  import Modal from '../components/Modal.svelte';
  import PanelHeader from '../components/PanelHeader.svelte';
  import Seg from '../components/Seg.svelte';
  import Switch from '../components/Switch.svelte';
  import { api, errorMessage } from '../lib/api';
  import { FORMAT_NAMES } from '../lib/format';
  import { refreshHealth } from '../lib/health.svelte';
  import { addGuard } from '../lib/router.svelte';
  import { phone } from '../lib/media.svelte';
  import { refreshSession } from '../lib/session.svelte';
  import { bottomBar, toast } from '../lib/toast.svelte';
  import type { ImageFormat, Settings, Visibility, WritableSettings } from '../lib/types';

  const ALL_FORMATS: ImageFormat[] = ['jpeg', 'png', 'gif', 'webp', 'avif'];
  const MB = 1024 * 1024;

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
  type NumberKey = 'maxMB' | 'maxMP' | 'retention' | 'perMinute' | 'idle' | 'absolute';

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

  let baseline = $state<Settings | null>(null);
  let phase = $state<'loading' | 'ready' | 'error'>('loading');
  let loadError = $state('');
  let form = $state<Form>({
    maxMB: '',
    maxMP: '',
    formats: { jpeg: false, png: false, gif: false, webp: false, avif: false },
    defaultVisibility: 'public',
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

  // What the form would send, in API units.
  const values = $derived<WritableSettings>({
    maxUploadBytes: Math.round(num(form.maxMB) * MB),
    maxPixels: Math.round(num(form.maxMP) * 1e6),
    allowedFormats: ALL_FORMATS.filter((f) => form.formats[f]),
    dedupe: form.dedupe,
    defaultVisibility: form.defaultVisibility,
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
      formats: values.allowedFormats.length === 0 ? '至少选择一种格式' : '',
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
    if (!baseline) return [] as (keyof WritableSettings)[];
    const base = baseline;
    return (Object.keys(values) as (keyof WritableSettings)[]).filter((k) => !same(values[k], base[k]));
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
      const s = await api.patchSettings(patch as Partial<WritableSettings>);
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
    leaveResolve?.(ok);
    leaveResolve = null;
  }
  // On phones the bar is fixed above the tab bar, so toasts have to clear it.
  (() => {
    bottomBar.active = phone.matches && dirty;
  });
  onDestroy(() => (bottomBar.active = false));

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
</script>

{#snippet numberField(key: NumberKey, id: string, unit: string)}
  <div class="input-unit" class:invalid={!!errors[key]}>
    <input
      {id}
      class="num"
      inputmode={key === 'maxMB' || key === 'maxMP' ? 'decimal' : 'numeric'}
      aria-invalid={errors[key] ? 'true' : undefined}
      bind:value={form[key]}
    />
    <span class="unit">{unit}</span>
  </div>
  {#if errors[key]}<span class="error-text">{errors[key]}</span>{/if}
{/snippet}

<div class="page">
  <PanelHeader title="设置" />
  <div class="page-body">
    <div class="adm-col">
      {#if phase === 'loading'}
        <p class="subtle">加载中…</p>
      {:else if phase === 'error' || !baseline}
        <div class="alert danger" role="alert">
          <Icon name="alert" />
          <div class="grow">加载失败：{loadError}</div>
          <button type="button" class="btn btn-secondary btn-sm" onclick={() => void load()}>重试</button>
        </div>
      {:else}
        <AdmSection title="上传">
          <AdmRow label="单文件上限" htmlFor="adm-max-mb">
            {@render numberField('maxMB', 'adm-max-mb', 'MB')}
          </AdmRow>
          <AdmRow label="像素上限" htmlFor="adm-max-px">
            {@render numberField('maxMP', 'adm-max-px', '百万像素')}
          </AdmRow>
          <AdmRow label="格式">
            <div class="adm-opts-row">
              {#each ALL_FORMATS as f (f)}
                <label class="adm-opt-inline">
                  <input type="checkbox" class="check" bind:checked={form.formats[f]} />
                  {FORMAT_NAMES[f]}
                </label>
              {/each}
            </div>
            {#if errors.formats}<span class="error-text">{errors.formats}</span>{/if}
          </AdmRow>
          <AdmRow label="频率上限" htmlFor="adm-rate">
            {@render numberField('perMinute', 'adm-rate', '次/分钟')}
          </AdmRow>
          <AdmRow label="重复检测">
            <div class="adm-value">
              <Switch checked={form.dedupe} label="重复检测" onchange={(v) => (form.dedupe = v)} />
            </div>
          </AdmRow>
        </AdmSection>

        <AdmSection title="图片">
          <AdmRow label="默认可见性">
            <Seg
              label="默认可见性"
              value={form.defaultVisibility}
              onchange={(v: Visibility) => (form.defaultVisibility = v)}
              options={[
                { key: 'public', label: '公开', icon: 'globe' },
                { key: 'private', label: '私有', icon: 'lock' },
              ]}
            />
          </AdmRow>
          <AdmRow label="回收站保留" htmlFor="adm-trash-days">
            {@render numberField('retention', 'adm-trash-days', '天')}
          </AdmRow>
        </AdmSection>

        <AdmSection title="会话">
          <AdmRow label="空闲超时" htmlFor="adm-idle">
            {@render numberField('idle', 'adm-idle', '分钟')}
          </AdmRow>
          <AdmRow label="最长有效" htmlFor="adm-abs">
            {@render numberField('absolute', 'adm-abs', '小时')}
          </AdmRow>
        </AdmSection>

        <AdmSection title="地址">
          <AdmRow label="公开地址">
            <div class="adm-value mono">{baseline.publicUrl || '—'}</div>
          </AdmRow>
          <AdmRow label="图片地址">
            <div class="adm-value mono">{baseline.imageUrl || '—'}</div>
          </AdmRow>
        </AdmSection>

        {#if dirty}
          <div class="adm-sticky" role="region" aria-label="未保存的更改">
            <span class="grow adm-pair" aria-live="polite">
              <span>{changed.length} 项更改未保存</span>
              {#if saveError}<span class="error-text">{saveError}</span>{/if}
            </span>
            <button type="button" class="btn btn-secondary" disabled={saving} onclick={discard}>放弃</button>
            <button type="button" class="btn btn-primary" disabled={saving || invalid} onclick={save}>保存</button>
          </div>
        {/if}
      {/if}
    </div>
  </div>
</div>
{#if leaveOpen}
  <Modal title="离开此页？" width="400px" onclose={() => answer(false)}>
    <p>有未保存的更改，离开后将丢失。</p>
    {#snippet footer()}
      <button type="button" class="btn btn-secondary" data-autofocus onclick={() => answer(false)}>留下</button>
      <button type="button" class="btn btn-danger-solid" onclick={() => answer(true)}>离开</button>
    {/snippet}
  </Modal>
{/if}
