<script lang="ts">
  import AuthPanel from '../components/AuthPanel.svelte';
  import Button from '../components/Button.svelte';
  import TextField from '../components/TextField.svelte';
  import { api, ApiError, errorMessage } from '../lib/api';
  import { refreshSession } from '../lib/session.svelte';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let busy = $state(false);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (busy) return;
    busy = true;
    error = '';
    try {
      await api.login(username.trim(), password);
      // The App gate sees the new session and navigates to ?next= (or the home route).
      await refreshSession();
    } catch (err) {
      if (err instanceof ApiError && err.code === 'setup-required') {
        await refreshSession();
      } else if (err instanceof ApiError && err.code === 'unauthorized') {
        error = '用户名或密码不正确';
      } else {
        error = errorMessage(err);
      }
      password = '';
    } finally {
      busy = false;
    }
  }
</script>

<AuthPanel title="登录" subtitle="登录后管理你的底片">
  <form onsubmit={submit} class="form" novalidate>
    <TextField label="用户名" bind:value={username} autocomplete="username" autocapitalize="none" spellcheck={false} autofocus required />
    <TextField label="密码" type="password" bind:value={password} autocomplete="current-password" required />
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <div class="actions">
      <Button type="submit" variant="primary" block disabled={busy || !username || !password}>登录</Button>
    </div>
  </form>
</AuthPanel>

<style>
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
</style>
