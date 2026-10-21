<script lang="ts">
  import CopyButton from '../components/CopyButton.svelte';
  import Icon from '../components/Icon.svelte';
  import Modal from '../components/Modal.svelte';
  import Seg from '../components/Seg.svelte';
  import { api, errorMessage } from '../lib/api';
  import { toast } from '../lib/toast.svelte';
  import type { Scope, Token } from '../lib/types';

  /** `finished` runs when the dialog closes after a token was created, so the list can refresh. */
  let { onclose, onfinished }: { onclose: () => void; onfinished: () => void } = $props();

  const SCOPE_LABEL: Record<Scope, string> = { upload: '上传', read: '只读', manage: '管理' };
  const SCOPES: { key: Scope; hint: string }[] = [
    { key: 'upload', hint: '上传图片，只能查看自己上传的图片' },
    { key: 'read', hint: '查看全部图片' },
    { key: 'manage', hint: '修改可见性、删除和恢复图片' },
  ];
  const EXPIRY = [
    { value: '', label: '永不过期' },
    { value: '30', label: '30 天' },
    { value: '90', label: '90 天' },
    { value: '365', label: '1 年' },
  ];

  let name = $state('');
  let scopes = $state<Record<Scope, boolean>>({ upload: true, read: false, manage: false });
  let days = $state('');
  let tried = $state(false);
  let busy = $state(false);
  let formError = $state('');
  let created = $state<{ token: Token; secret: string } | null>(null);
  let kind = $state<'curl' | 'picgo' | 'sharex'>('curl');

  const chosen = $derived(SCOPES.filter((s) => scopes[s.key]).map((s) => s.key));
  const nameErr = $derived(name.trim() ? (name.trim().length > 80 ? '名称最多 80 个字符' : '') : '请输入名称');
  const scopeErr = $derived(chosen.length > 0 ? '' : '请选择至少一项权限');

  async function submit() {
    tried = true;
    formError = '';
    if (nameErr || scopeErr || busy) return;
    busy = true;
    try {
      created = await api.createToken({ name: name.trim(), scopes: chosen, expiresInDays: days === '' ? null : Number(days) });
    } catch (e) {
      formError = errorMessage(e);
    } finally {
      busy = false;
    }
  }

  function finish() {
    const had = created !== null;
    created = null;
    onclose();
    if (had) onfinished();
  }

  // Examples use the origin the page is served from.
  const endpoint = `${location.origin}/api/v1/images`;
  const secret = $derived(created?.secret ?? '');
  const snippet = $derived.by(() => {
    if (kind === 'curl') return `curl -H "Authorization: Bearer ${secret}" -F file=@shot.png ${endpoint}`;
    if (kind === 'picgo') {
      return [`API 地址: ${endpoint}`, 'POST 参数名: file', 'JSON 路径: url', `自定义请求头: ${JSON.stringify({ Authorization: `Bearer ${secret}` })}`].join('\n');
    }
    return [`RequestURL: ${endpoint}`, 'FileFormName: file', 'URL: {json:url}', `Headers: Authorization: Bearer ${secret}`].join('\n');
  });

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
    toast('已下载 PicBerth.sxcu');
  }
</script>

{#if created}
  <Modal title="令牌已创建" width="520px" dismissible={false} onclose={finish}>
    <div class="adm-form">
      <div class="alert warning">
        <Icon name="alert" />
        <div class="grow">关闭后将无法再次查看此令牌</div>
      </div>
      <div class="field">
        <label for="adm-secret">令牌</label>
        <div class="adm-secret">
          <input id="adm-secret" class="input" readonly value={created.secret} onfocus={(e) => e.currentTarget.select()} />
          <CopyButton text={created.secret} />
        </div>
      </div>
      <div class="adm-form" style="gap: 10px">
        <div>
          <Seg
            label="使用方式"
            value={kind}
            onchange={(k: 'curl' | 'picgo' | 'sharex') => (kind = k)}
            options={[
              { key: 'curl', label: 'curl' },
              { key: 'picgo', label: 'PicGo' },
              { key: 'sharex', label: 'ShareX' },
            ]}
          />
        </div>
        {#if kind === 'sharex'}
          <div>
            <button type="button" class="btn btn-secondary btn-sm" onclick={downloadSxcu}>
              <Icon name="download" />
              下载 .sxcu
            </button>
          </div>
        {/if}
        <pre class="adm-pre">{snippet}</pre>
      </div>
    </div>
    {#snippet footer()}
      <button type="button" class="btn btn-primary" data-autofocus onclick={finish}>完成</button>
    {/snippet}
  </Modal>
{:else}
  <Modal title="新建令牌" width="520px" {onclose}>
    <form
      class="adm-form"
      id="token-form"
      novalidate
      onsubmit={(e) => {
        e.preventDefault();
        void submit();
      }}
    >
      <div class="field">
        <label for="adm-tk-name">名称</label>
        <input
          id="adm-tk-name"
          class="input"
          class:adm-invalid={tried && nameErr}
          aria-invalid={tried && nameErr ? 'true' : undefined}
          bind:value={name}
          maxlength={120}
          autocomplete="off"
          data-autofocus
        />
        {#if tried && nameErr}<span class="error-text">{nameErr}</span>{/if}
      </div>
      <fieldset class="field" style="border: 0; padding: 0; min-width: 0">
        <legend style="font-size: 13px; font-weight: 500; padding: 0; margin-bottom: 6px">权限</legend>
        <div class="adm-opts">
          {#each SCOPES as o (o.key)}
            <label class="adm-opt">
              <input type="checkbox" class="check" bind:checked={scopes[o.key]} />
              <span>
                <strong>{SCOPE_LABEL[o.key]}</strong>
                <span class="hint">{o.hint}</span>
              </span>
            </label>
          {/each}
        </div>
        {#if tried && scopeErr}<span class="error-text">{scopeErr}</span>{/if}
      </fieldset>
      <div class="field">
        <label for="adm-tk-exp">有效期</label>
        <select id="adm-tk-exp" class="select" bind:value={days}>
          {#each EXPIRY as o (o.value)}
            <option value={o.value}>{o.label}</option>
          {/each}
        </select>
      </div>
      {#if formError}<span class="error-text" role="alert">{formError}</span>{/if}
    </form>
    {#snippet footer()}
      <button type="button" class="btn btn-secondary" onclick={onclose}>取消</button>
      <button type="submit" form="token-form" class="btn btn-primary" disabled={busy}>创建</button>
    {/snippet}
  </Modal>
{/if}
