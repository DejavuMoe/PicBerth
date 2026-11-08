<script lang="ts">
  import { api, ApiError, errorMessage } from '../lib/api';
  import { refreshSession } from '../lib/session.svelte';
  import Icon from '../components/Icon.svelte';

  let code = $state('');
  let username = $state('admin');
  let password = $state('');
  let confirm = $state('');
  let submitted = $state(false);
  let busy = $state(false);
  let formError = $state('');

  // Setup codes are 12 Crockford base32 characters (no I, L, O, U), shown as XXXX-XXXX-XXXX.
  const INVALID = /[^0-9A-HJKMNP-TV-Z]/g;
  const CODE_LEN = 12;

  function formatCode(raw: string): string {
    const chars = raw.toUpperCase().replace(INVALID, '').slice(0, CODE_LEN);
    return chars.match(/.{1,4}/g)?.join('-') ?? '';
  }

  // Upper-case, drop invalid characters and keep the dashes in place while typing or pasting,
  // without letting the caret jump to the end.
  function oncodeinput(e: Event & { currentTarget: HTMLInputElement }) {
    const el = e.currentTarget;
    const caret = el.selectionStart ?? el.value.length;
    const before = el.value.slice(0, caret).toUpperCase().replace(INVALID, '').length;
    const next = formatCode(el.value);
    code = next;
    if (el.value !== next) el.value = next;
    let pos = 0;
    let seen = 0;
    while (pos < next.length && seen < Math.min(before, CODE_LEN)) {
      if (next[pos] !== '-') seen++;
      pos++;
    }
    el.setSelectionRange(pos, pos);
  }

  const codeComplete = $derived(code.replace(/-/g, '').length === CODE_LEN);

  const errors = $derived({
    code: !code.trim()
      ? '请输入初始化代码'
      : codeComplete
        ? null
        : '初始化代码为 12 位',
    username: /^[A-Za-z0-9._-]{1,64}$/.test(username.trim()) ? null : '用户名为 1–64 位字母、数字或 . _ -',
    password: password.length >= 10 && password.length <= 256 ? null : '密码需要 10–256 个字符',
    confirm: confirm === password ? null : '两次输入的密码不一致',
  });
  const valid = $derived(!errors.code && !errors.username && !errors.password && !errors.confirm);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    submitted = true;
    formError = '';
    if (!valid || busy) return;
    busy = true;
    try {
      await api.setup(code.trim().toUpperCase(), username.trim(), password);
      await refreshSession();
    } catch (err) {
      if (err instanceof ApiError && err.code === 'setup-done') {
        formError = '管理员已经创建，请直接登录';
        await refreshSession();
      } else if (err instanceof ApiError && err.code === 'forbidden') {
        formError = '初始化代码不正确';
      } else {
        formError = errorMessage(err);
      }
    } finally {
      busy = false;
    }
  }
</script>

<div class="auth">
  <div class="auth-box">
    <div class="auth-brand">
      <span class="brand-mark" aria-hidden="true"></span>
      PicBerth
    </div>
    <h1>初始化</h1>
    <form class="auth-form" onsubmit={submit} novalidate>
      <div class="field">
        <label for="setup-code">初始化代码</label>
        <!-- svelte-ignore a11y_autofocus -->
        <input
          id="setup-code"
          class="input mono"
          value={code}
          oninput={oncodeinput}
          autocomplete="off"
          autocapitalize="characters"
          spellcheck="false"
          maxlength={14}
          placeholder="XXXX-XXXX-XXXX"
          aria-invalid={submitted && errors.code ? 'true' : undefined}
          autofocus
        />
        {#if submitted && errors.code}
          <span class="field-error">{errors.code}</span>
        {:else}
          <span class="s muted">在服务器启动日志中</span>
        {/if}
      </div>
      <div class="field">
        <label for="setup-user">用户名</label>
        <input id="setup-user" class="input" bind:value={username} autocomplete="username" autocapitalize="none" spellcheck="false" aria-invalid={submitted && errors.username ? 'true' : undefined} />
        {#if submitted && errors.username}<span class="field-error">{errors.username}</span>{/if}
      </div>
      <div class="field">
        <label for="setup-pass">密码</label>
        <input id="setup-pass" class="input" type="password" bind:value={password} autocomplete="new-password" aria-invalid={submitted && errors.password ? 'true' : undefined} />
        {#if submitted && errors.password}<span class="field-error">{errors.password}</span>{/if}
      </div>
      <div class="field">
        <label for="setup-confirm">确认密码</label>
        <input id="setup-confirm" class="input" type="password" bind:value={confirm} autocomplete="new-password" aria-invalid={submitted && errors.confirm ? 'true' : undefined} />
        {#if submitted && errors.confirm}<span class="field-error">{errors.confirm}</span>{/if}
      </div>
      {#if formError}
        <div class="alert" role="alert">
          <Icon name="alert" />
          <span>{formError}</span>
        </div>
      {/if}
      <button type="submit" class="btn btn-primary btn-block" disabled={busy}>{busy ? '正在创建…' : '创建管理员'}</button>
    </form>
  </div>
</div>
