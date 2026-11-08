<script lang="ts">
  import Icon from '../Icon.svelte';
  import Modal from '../Modal.svelte';
  import Select from '../Select.svelte';
  import { api, errorMessage } from '../../lib/api';
  import type { Scope, Token } from '../../lib/types';

  let { onclose, oncreated }: { onclose: () => void; oncreated: (token: Token, secret: string) => void } = $props();

  const SCOPES: { key: Scope; label: string }[] = [
    { key: 'upload', label: '上传' },
    { key: 'read', label: '只读' },
    { key: 'manage', label: '管理' },
  ];
  const EXPIRY = [
    { value: '', label: '永不过期' },
    { value: '30', label: '30 天' },
    { value: '90', label: '90 天' },
    { value: '365', label: '1 年' },
  ];

  let name = $state('');
  let scopes = $state<Scope[]>(['upload']);
  let days = $state('');
  let tried = $state(false);
  let busy = $state(false);
  let formError = $state('');

  const nameErr = $derived(!name.trim() ? '请输入名称' : name.trim().length > 80 ? '名称最多 80 个字符' : '');
  const scopeErr = $derived(scopes.length > 0 ? '' : '请选择至少一项权限');

  function toggle(key: Scope) {
    scopes = scopes.includes(key) ? scopes.filter((s) => s !== key) : [...scopes, key];
  }

  async function submit() {
    tried = true;
    formError = '';
    if (nameErr || scopeErr || busy) return;
    busy = true;
    try {
      const { token, secret } = await api.createToken({
        name: name.trim(),
        scopes,
        expiresInDays: days === '' ? null : Number(days),
      });
      oncreated(token, secret);
    } catch (e) {
      formError = errorMessage(e);
      busy = false;
    }
  }
</script>

<Modal title="新建令牌" {onclose}>
  <form
    class="auth-form"
    id="token-form"
    novalidate
    onsubmit={(e) => {
      e.preventDefault();
      void submit();
    }}
  >
    <div class="field">
      <label for="tk-name">名称</label>
      <input
        id="tk-name"
        class="input"
        aria-invalid={tried && nameErr ? 'true' : undefined}
        bind:value={name}
        maxlength={120}
        autocomplete="off"
        data-autofocus
      />
      {#if tried && nameErr}<span class="field-error">{nameErr}</span>{/if}
    </div>
    <fieldset class="field" style="border: 0; margin: 0; padding: 0;">
      <legend class="label medium" style="padding: 0; margin-bottom: 6px;">权限</legend>
      <div class="scope-opts">
        {#each SCOPES as s (s.key)}
          {@const on = scopes.includes(s.key)}
          <button type="button" class="scope-opt" role="checkbox" aria-checked={on} onclick={() => toggle(s.key)}>
            <span class="check" data-on={String(on)} aria-hidden="true">
              {#if on}<Icon name="check" />{/if}
            </span>
            {s.label}
          </button>
        {/each}
      </div>
      {#if tried && scopeErr}<span class="field-error">{scopeErr}</span>{/if}
    </fieldset>
    <div class="field">
      <span class="label">有效期</span>
      <Select label="有效期" value={days} onchange={(v) => (days = v)} options={EXPIRY} width={160} />
    </div>
    {#if formError}<span class="field-error" role="alert">{formError}</span>{/if}
  </form>
  {#snippet footer()}
    <button type="button" class="btn btn-secondary" onclick={onclose}>取消</button>
    <button type="submit" form="token-form" class="btn btn-primary" disabled={busy}>创建</button>
  {/snippet}
</Modal>
