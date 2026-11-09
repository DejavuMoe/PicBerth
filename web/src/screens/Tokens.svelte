<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import Icon from '../components/Icon.svelte';
  import TokenCreate from '../components/admin/TokenCreate.svelte';
  import TokenReveal from '../components/admin/TokenReveal.svelte';
  import { api, errorMessage } from '../lib/api';
  import { formatAbsolute, formatDate, formatRelative } from '../lib/format';
  import { toast } from '../lib/toast.svelte';
  import type { Scope, Token } from '../lib/types';

  const SCOPE_LABEL: Record<Scope, string> = { upload: '上传', manage: '管理' };

  let tokens = $state<Token[]>([]);
  let phase = $state<'loading' | 'ready' | 'error'>('loading');
  let errorText = $state('');
  let creating = $state(false);
  let revealed = $state<string | null>(null);
  // Revoking cannot be undone, so the button asks once more in place: 吊销 turns into 确认吊销 for a few seconds.
  let arming = $state<number | null>(null);
  let disarm = 0;
  let disblur = 0;
  onDestroy(() => {
    clearTimeout(disarm);
    clearTimeout(disblur);
  });

  async function load() {
    try {
      tokens = (await api.listTokens()).items;
      phase = 'ready';
    } catch (e) {
      errorText = errorMessage(e);
      phase = 'error';
    }
  }
  onMount(() => void load());

  async function revoke(t: Token) {
    clearTimeout(disarm);
    if (arming !== t.id) {
      arming = t.id;
      disarm = window.setTimeout(() => (arming = null), 4000);
      return;
    }
    arming = null;
    try {
      await api.revokeToken(t.id);
      toast('令牌已吊销');
      await load();
    } catch (e) {
      toast(errorMessage(e));
    }
  }

  // Prefixes are stored without the ellipsis; it marks "the rest is secret".
  const shownPrefix = (p: string) => (p.endsWith('…') ? p : `${p}…`);
  const scopeNames = (t: Token) => t.scopes.map((s) => SCOPE_LABEL[s] ?? s).join('、');
  const expired = (t: Token) => !!t.expiresAt && new Date(t.expiresAt).getTime() < Date.now();
  const expiry = (t: Token) =>
    t.revokedAt ? '已吊销' : !t.expiresAt ? '永不过期' : expired(t) ? '已过期' : formatDate(t.expiresAt);
  const lastUsed = (t: Token) => (t.lastUsedAt ? formatRelative(t.lastUsedAt) : '从未使用');

  const live = $derived(tokens.filter((t) => !t.revokedAt));
  const dead = $derived(tokens.filter((t) => !!t.revokedAt));
</script>

{#snippet row(t: Token)}
  <div class="list-row tok-row" class:revoked={!!t.revokedAt}>
    <div class="tok-name">
      <span class="n ellipsis" title={t.name}>{t.name}</span>
      <span class="mono">{shownPrefix(t.prefix)}</span>
      <span class="tok-sub">
        <span>{scopeNames(t)}</span>
        <span>{lastUsed(t)}</span>
        <span>{expiry(t)}</span>
      </span>
    </div>
    <span class="c-scope ellipsis">{scopeNames(t)}</span>
    <span class="c-used muted num" title={t.lastUsedAt ? formatAbsolute(t.lastUsedAt) : undefined}>
      {lastUsed(t)}
    </span>
    <span class="c-exp muted num">{expiry(t)}</span>
    {#if t.revokedAt}
      <span></span>
    {:else}
      <button
        type="button"
        class="btn btn-sm {arming === t.id ? 'btn-danger-armed' : 'btn-ghost'}"
        aria-label={arming === t.id ? `确认吊销 ${t.name}` : `吊销 ${t.name}`}
        onclick={() => void revoke(t)}
        onblur={() => {
          if (arming !== t.id) return;
          clearTimeout(disblur);
          disblur = window.setTimeout(() => {
            if (arming === t.id) arming = null;
          }, 150);
        }}
      >
        {arming === t.id ? '确认吊销' : '吊销'}
      </button>
    {/if}
  </div>
{/snippet}

<div class="panel-main">
  <div class="head">
    <div class="head-title">
      <h1>令牌</h1>
    </div>
    <a class="btn btn-ghost" href="/docs/api" target="_blank" rel="noopener">
      API 文档
      <Icon name="external" size="s" />
    </a>
    <button type="button" class="btn btn-primary" onclick={() => (creating = true)}>
      <Icon name="plus" />
      新建令牌
    </button>
  </div>
  <div class="scroll">
    <div class="page" style="max-width: 960px;">
      {#if phase === 'loading'}
        <div class="list" aria-busy="true">
          {#each [0, 1, 2] as i (i)}
            <div class="list-row tok-row">
              <div class="skeleton" style="height: 10px; width: 60%;"></div>
            </div>
          {/each}
        </div>
      {:else if phase === 'error'}
        <div class="state" role="alert" style="padding: 64px 0;">
          <span class="t">令牌加载失败</span>
          <span class="d">{errorText}</span>
          <button type="button" class="btn btn-secondary" onclick={() => void load()}>重试</button>
        </div>
      {:else if tokens.length === 0}
        <div class="state" style="padding: 64px 0;">
          <span class="t">还没有令牌</span>
          <span class="d">PicGo、ShareX 等工具通过令牌上传图片</span>
        </div>
      {:else}
        <div class="list" role="table" aria-label="令牌">
          <div class="list-row list-head tok-row" role="row">
            <span>名称</span>
            <span class="c-scope">权限</span>
            <span class="c-used">最近使用</span>
            <span class="c-exp">到期</span>
            <span></span>
          </div>
          {#each live as t (t.id)}
            {@render row(t)}
          {/each}
          {#each dead as t (t.id)}
            {@render row(t)}
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>

{#if creating}
  <TokenCreate
    onclose={() => (creating = false)}
    oncreated={(token, secret) => {
      creating = false;
      revealed = secret;
      tokens = [token, ...tokens];
    }}
  />
{/if}
{#if revealed !== null}
  <TokenReveal
    secret={revealed}
    ondone={() => {
      revealed = null;
      void load();
    }}
  />
{/if}
