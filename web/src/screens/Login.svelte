<script lang="ts">
  import AuthCard from '../components/AuthCard.svelte';
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

<AuthCard title="登录">
  <form class="auth-form" onsubmit={submit} novalidate>
    <div class="field">
      <label for="login-user">用户名</label>
      <!-- svelte-ignore a11y_autofocus -->
      <input id="login-user" class="input" bind:value={username} autocomplete="username" autocapitalize="none" spellcheck="false" autofocus required />
    </div>
    <div class="field">
      <label for="login-pass">密码</label>
      <input id="login-pass" class="input" type="password" bind:value={password} autocomplete="current-password" required />
    </div>
    {#if error}<p class="error-text" role="alert">{error}</p>{/if}
    <button type="submit" class="btn btn-primary btn-block" disabled={busy || !username || !password}>{busy ? '正在登录…' : '登录'}</button>
  </form>
</AuthCard>
