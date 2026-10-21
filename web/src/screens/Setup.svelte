<script lang="ts">
  import AuthPanel from '../components/AuthPanel.svelte';
  import Button from '../components/Button.svelte';
  import TextField from '../components/TextField.svelte';
  import { api, ApiError, errorMessage } from '../lib/api';
  import { refreshSession } from '../lib/session.svelte';

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
  // 1: look the code up in the server log, 2: fill in the administrator account.
  const step = $derived(codeComplete ? 2 : 1);
  const steps = ['在服务器日志中找到初始化代码', '创建管理员账户', '开始冲洗照片'];

  const errors = $derived({
    code: !code.trim()
      ? '请输入服务器日志中的初始化代码'
      : codeComplete
        ? null
        : '初始化代码共 12 位，格式为 XXXX-XXXX-XXXX',
    username: /^[A-Za-z0-9._-]{1,64}$/.test(username.trim()) ? null : '1–64 个字符，可用字母、数字、点、下划线和连字符',
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

<AuthPanel title="初始化" subtitle="第一次启动，需要创建管理员">
  <ol class="steps" aria-label="初始化步骤">
    {#each steps as text, i (i)}
      <li class="step" class:current={step === i + 1} aria-current={step === i + 1 ? 'step' : undefined}>
        <span class="num mono">{String(i + 1).padStart(2, '0')}</span>
        <span>{text}</span>
      </li>
    {/each}
  </ol>

  <form onsubmit={submit} class="form" novalidate>
    <div class="codefield">
      <TextField
        label="初始化代码"
        bind:value={code}
        oninput={oncodeinput}
        code
        autocomplete="off"
        autocapitalize="characters"
        spellcheck={false}
        placeholder="XXXX-XXXX-XXXX"
        error={submitted ? errors.code : null}
        autofocus
      />
    </div>
    <TextField
      label="用户名"
      bind:value={username}
      autocomplete="username"
      autocapitalize="none"
      spellcheck={false}
      error={submitted ? errors.username : null}
    />
    <TextField
      label="密码"
      type="password"
      bind:value={password}
      autocomplete="new-password"
      hint="至少 10 个字符"
      error={submitted ? errors.password : null}
    />
    <TextField
      label="确认密码"
      type="password"
      bind:value={confirm}
      autocomplete="new-password"
      error={submitted ? errors.confirm : null}
    />
    {#if formError}<p class="error" role="alert">{formError}</p>{/if}
    <div class="actions">
      <Button type="submit" variant="primary" block disabled={busy}>创建管理员</Button>
    </div>
  </form>
</AuthPanel>

<style>
  .steps {
    display: flex;
    flex-direction: column;
    gap: var(--s1);
    margin-bottom: var(--s5);
    padding-bottom: var(--s4);
    border-bottom: 1px solid var(--line);
  }
  .step {
    display: flex;
    align-items: baseline;
    gap: var(--s3);
    color: var(--ink-3);
    font-size: var(--t-13);
    transition: color var(--dur) var(--ease);
  }
  .step.current {
    color: var(--ink);
    font-weight: 600;
  }
  .num {
    flex: none;
    width: 2ch;
    font-size: var(--t-12);
  }

  .form {
    display: flex;
    flex-direction: column;
    gap: var(--s4);
  }
  .actions {
    margin-top: var(--s2);
  }
  .error {
    color: var(--danger);
    font-size: var(--t-13);
  }

  /* The code is the one thing to read off a server log: large, spaced mono. */
  .codefield :global(.control) {
    height: 56px;
  }
  .codefield :global(input.code) {
    font-size: 20px;
    font-weight: 500;
    letter-spacing: 0.2em;
  }
  .codefield :global(input.code::placeholder) {
    font-weight: 400;
  }
</style>
