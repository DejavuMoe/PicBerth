<script lang="ts">
  import { untrack } from 'svelte';
  import Button from '../components/Button.svelte';
  import Checkbox from '../components/Checkbox.svelte';
  import CopyButton from '../components/CopyButton.svelte';
  import Dialog from '../components/Dialog.svelte';
  import Icon from '../components/Icon.svelte';
  import Select from '../components/Select.svelte';
  import Tabs from '../components/Tabs.svelte';
  import TextField from '../components/TextField.svelte';
  import Ticket from '../components/Ticket.svelte';
  import { api, errorMessage } from '../lib/api';
  import type { Scope, Token } from '../lib/types';

  interface Props {
    open: boolean;
    /** Called when the dialog closes after a token was created, so the list can refresh. */
    onfinished: () => void;
  }
  let { open = $bindable(false), onfinished }: Props = $props();

  const SCOPES: { value: Scope; label: string; help: string }[] = [
    { value: 'upload', label: '上传', help: '上传图片，只能查看自己上传的图片' },
    { value: 'read', label: '只读', help: '查看全部图片，包括私有图片的内容' },
    { value: 'manage', label: '管理', help: '读取、上传，并可修改可见性、删除、恢复和永久删除' },
  ];
  const EXPIRY = [
    { value: '', label: '永不过期' },
    { value: '30', label: '30 天' },
    { value: '90', label: '90 天' },
    { value: '365', label: '1 年' },
  ];

  let step = $state<'form' | 'reveal'>('form');
  let name = $state('');
  let scopes = $state<Record<Scope, boolean>>({ upload: true, read: false, manage: false });
  let expiry = $state('');
  let submitted = $state(false);
  let busy = $state(false);
  let formError = $state('');
  let created = $state<{ token: Token; secret: string } | null>(null);
  let tab = $state<'curl' | 'picgo' | 'sharex'>('curl');

  // A fresh form each time the dialog opens; the secret never outlives the dialog.
  $effect(() => {
    if (open) {
      untrack(() => {
        step = 'form';
        name = '';
        scopes = { upload: true, read: false, manage: false };
        expiry = '';
        submitted = false;
        formError = '';
        created = null;
        tab = 'curl';
      });
    }
  });

  const chosen = $derived(SCOPES.filter((s) => scopes[s.value]).map((s) => s.value));
  const errors = $derived({
    name: name.trim().length === 0 ? '请输入名称' : name.trim().length > 80 ? '名称最多 80 个字符' : null,
    scopes: chosen.length === 0 ? '至少选择一项权限' : null,
  });
  const createdScopeLabels = $derived(
    created ? created.token.scopes.map((s) => SCOPES.find((x) => x.value === s)?.label ?? s) : [],
  );

  async function create(e: SubmitEvent) {
    e.preventDefault();
    submitted = true;
    formError = '';
    if (errors.name || errors.scopes || busy) return;
    busy = true;
    try {
      created = await api.createToken({
        name: name.trim(),
        scopes: chosen,
        expiresInDays: expiry === '' ? null : Number(expiry),
      });
      step = 'reveal';
    } catch (err) {
      formError = errorMessage(err);
    } finally {
      busy = false;
    }
  }

  function finish() {
    open = false;
    handleClosed();
  }
  function handleClosed() {
    const had = created !== null;
    created = null;
    if (had) onfinished();
  }

  // Examples always use the origin the user is looking at.
  const origin = location.origin;
  const endpoint = `${origin}/api/v1/images`;
  const secret = $derived(created?.secret ?? '');
  const curl = $derived(`curl -H "Authorization: Bearer ${secret}" -F file=@shot.png ${endpoint}`);
  const picgoHeaders = $derived(JSON.stringify({ Authorization: `Bearer ${secret}` }));

  function downloadSxcu() {
    const config = {
      Version: '15.0.0',
      Name: `PicBerth (${location.host})`,
      DestinationType: 'ImageUploader',
      RequestMethod: 'POST',
      RequestURL: endpoint,
      Headers: { Authorization: `Bearer ${secret}` },
      Body: 'MultipartFormData',
      FileFormName: 'file',
      URL: '{json:url}',
      ErrorMessage: '{json:detail}',
    };
    const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
    const href = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = href;
    a.download = 'PicBerth.sxcu';
    document.body.append(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(href), 1000);
  }
</script>

{#snippet code(value: string, label: string)}
  <div class="code">
    <pre class="mono">{value}</pre>
    <CopyButton iconOnly size="sm" {value} {label} />
  </div>
{/snippet}

{#snippet kv(label: string, value: string)}
  <div class="kv">
    <p class="k">{label}</p>
    {@render code(value, `复制${label}`)}
  </div>
{/snippet}

<Dialog
  bind:open
  title={step === 'form' ? '新建令牌' : '令牌已创建'}
  width={step === 'form' ? '520px' : '640px'}
  dismissible={step === 'form'}
  onclose={handleClosed}
>
  {#if step === 'form'}
    <form id="token-form" class="form" onsubmit={create} novalidate>
      <TextField
        label="名称"
        bind:value={name}
        maxlength={120}
        placeholder="例如：笔记本上的 PicGo"
        autocomplete="off"
        error={submitted ? errors.name : null}
        autofocus
      />
      <fieldset class="scopes">
        <legend>权限</legend>
        <div class="scope-list">
          {#each SCOPES as s (s.value)}
            <div class="scope">
              <Checkbox
                label={s.label}
                description={s.help}
                checked={scopes[s.value]}
                onchange={(on) => (scopes[s.value] = on)}
              />
            </div>
          {/each}
        </div>
        {#if submitted && errors.scopes}<p class="err">{errors.scopes}</p>{/if}
      </fieldset>
      <Select label="有效期" options={EXPIRY} bind:value={expiry} block />
      {#if formError}<p class="err" role="alert">{formError}</p>{/if}
    </form>
  {:else if created}
    <div class="reveal">
      <Ticket name={created.token.name} scopes={createdScopeLabels} secret={created.secret} />
      <p class="warn"><Icon name="warning" /> 关闭后将无法再次查看此令牌</p>

      <h3 class="section-title head">接入示例</h3>
      <Tabs
        label="接入方式"
        idPrefix="tok"
        value={tab}
        onchange={(v) => (tab = v)}
        tabs={[
          { value: 'curl', label: 'curl' },
          { value: 'picgo', label: 'PicGo' },
          { value: 'sharex', label: 'ShareX' },
        ]}
      />
      <div class="panel" role="tabpanel" id="tok-panel" aria-labelledby="tok-tab-{tab}">
        {#if tab === 'curl'}
          {@render code(curl, '复制 curl 命令')}
        {:else if tab === 'picgo'}
          <p class="help">在 PicGo 的 web-uploader 图床设置中填写：</p>
          {@render kv('API 地址', endpoint)}
          {@render kv('POST 参数名', 'file')}
          {@render kv('JSON 路径', 'url')}
          {@render kv('自定义请求头', picgoHeaders)}
        {:else}
          <p class="help">下载配置文件，在 ShareX 的“自定义上传器设置”中选择“导入”→“从文件”。</p>
          <div class="dl">
            <Button icon="download" onclick={downloadSxcu}>下载 .sxcu</Button>
          </div>
          {@render kv('RequestURL', endpoint)}
          {@render kv('FileFormName', 'file')}
          {@render kv('URL', '{json:url}')}
          {@render kv('ErrorMessage', '{json:detail}')}
        {/if}
      </div>
    </div>
  {/if}

  {#snippet footer()}
    {#if step === 'form'}
      <Button onclick={() => (open = false)}>取消</Button>
      <Button type="submit" form="token-form" variant="primary" disabled={busy}>创建</Button>
    {:else}
      <Button variant="primary" onclick={finish}>完成</Button>
    {/if}
  {/snippet}
</Dialog>

<style>
  .form,
  .reveal {
    display: flex;
    flex-direction: column;
    gap: var(--s4);
  }
  .scopes {
    min-width: 0;
    padding: 0;
    border: 0;
  }
  legend {
    margin-bottom: var(--s2);
    padding: 0;
    color: var(--ink-2);
    font-size: var(--t-13);
    font-weight: 500;
  }
  .scope-list {
    border-top: 1px solid var(--line);
  }
  .scope {
    padding: var(--s3) 0;
    border-bottom: 1px solid var(--line);
  }
  /* The whole row is the hit area, not just the label text. */
  .scope :global(.cb) {
    display: flex;
    width: 100%;
  }
  .scope :global(.name) {
    font-weight: 600;
  }
  .scopes > .err {
    margin-top: var(--s2);
  }
  .err {
    color: var(--danger);
    font-size: var(--t-13);
  }

  .reveal {
    gap: var(--s3);
  }
  .warn {
    display: flex;
    align-items: center;
    gap: var(--s2);
    color: var(--warn);
    font-size: var(--t-13);
    font-weight: 550;
  }
  .head {
    margin-top: var(--s4);
  }
  .panel {
    display: flex;
    flex-direction: column;
    gap: var(--s3);
    padding-top: var(--s1);
  }
  .help {
    color: var(--ink-2);
    font-size: var(--t-13);
  }
  .dl {
    display: flex;
  }
  .kv {
    display: flex;
    flex-direction: column;
    gap: var(--s1);
    min-width: 0;
  }
  .k {
    color: var(--ink-3);
    font-size: var(--t-12);
  }
  .code {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: start;
    gap: var(--s1);
    padding: var(--s2) var(--s1) var(--s2) var(--s3);
    border-radius: 2px;
    background: var(--surface-2);
  }
  .code pre {
    margin: 0;
    padding-top: 4px;
    font-family: var(--font-mono);
    font-size: 12.5px;
    line-height: 1.6;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }
</style>
