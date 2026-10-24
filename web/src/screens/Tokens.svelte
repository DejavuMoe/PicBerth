<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from '../components/Icon.svelte';
  import PanelHeader from '../components/PanelHeader.svelte';
  import { api, errorMessage } from '../lib/api';
  import { formatAbsolute, formatDate, formatRelative } from '../lib/format';
  import { phone } from '../lib/media.svelte';
  import { toast } from '../lib/toast.svelte';
  import type { Scope, Token } from '../lib/types';
  import TokenDialog from './TokenDialog.svelte';

  const SCOPE_LABEL: Record<Scope, string> = { upload: '上传', read: '只读', manage: '管理' };

  let tokens = $state<Token[]>([]);
  let phase = $state<'loading' | 'ready' | 'error'>('loading');
  let errorText = $state('');
  let creating = $state(false);
  // Revoking cannot be undone, so the button asks once more in place: 吊销 turns into 确认吊销 for a few seconds.
  let arming = $state<number | null>(null);
  let disarm = 0;
  onMount(() => () => clearTimeout(disarm));

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
  const expired = (t: Token) => !!t.expiresAt && new Date(t.expiresAt).getTime() < Date.now();
</script>

{#snippet scopeBadges(scopes: Scope[])}
  <span class="adm-badges">
    {#each scopes as s (s)}
      <span class="badge" class:badge-warning={s === 'manage'}>{SCOPE_LABEL[s] ?? s}</span>
    {/each}
  </span>
{/snippet}

{#snippet expiry(t: Token)}
  {#if t.revokedAt}
    <span class="badge">已吊销</span>
  {:else if !t.expiresAt}
    <span>永不过期</span>
  {:else if expired(t)}
    <span class="badge badge-danger">已过期</span>
  {:else}
    <span class="num" title={formatAbsolute(t.expiresAt)}>{formatDate(t.expiresAt)}</span>
  {/if}
{/snippet}

{#snippet lastUsed(t: Token)}
  {#if t.lastUsedAt}
    <span title={formatAbsolute(t.lastUsedAt)}>{formatRelative(t.lastUsedAt)}</span>
  {:else}
    <span class="subtle">从未使用</span>
  {/if}
{/snippet}

{#snippet nameCell(t: Token)}
  <span class="tk-name">
    <span class="name truncate" title={t.name}>{t.name}</span>
    <span class="mono">{shownPrefix(t.prefix)}</span>
  </span>
{/snippet}

{#snippet revokeButton(t: Token)}
  {#if !t.revokedAt}
    <button
      type="button"
      class="btn btn-sm adm-revoke {arming === t.id ? 'btn-danger-solid armed' : 'btn-danger-ghost'}"
      aria-label={arming === t.id ? `确认吊销「${t.name}」` : `吊销「${t.name}」`}
      onclick={() => void revoke(t)}
      onblur={() => arming === t.id && (arming = null)}
    >
      {arming === t.id ? '确认吊销' : '吊销'}
    </button>
  {/if}
{/snippet}

{#snippet newButton()}
  <button type="button" class="btn btn-primary" onclick={() => (creating = true)}>
    <Icon name="plus" />
    新建令牌
  </button>
{/snippet}

<div class="page">
  <PanelHeader title="令牌">
    {#if phase === 'ready' && tokens.length > 0}{@render newButton()}{/if}
  </PanelHeader>
  <div class="page-body">
    {#if phase === 'loading'}
      <div class="adm-col"><p class="subtle">加载中…</p></div>
    {:else if phase === 'error'}
      <div class="adm-col">
        <div class="alert danger" role="alert">
          <Icon name="alert" />
          <div class="grow">加载失败：{errorText}</div>
          <button type="button" class="btn btn-secondary btn-sm" onclick={() => void load()}>重试</button>
        </div>
      </div>
    {:else if tokens.length === 0}
      <div class="empty">
        <Icon name="key" size={32} />
        <strong>还没有令牌</strong>
        <div class="actions">{@render newButton()}</div>
      </div>
    {:else}
      <div class="adm-col">
        {#if phone.matches}
          <div class="adm-trows">
            {#each tokens as t (t.id)}
              <div class="adm-trow" class:adm-revoked={!!t.revokedAt}>
                <div class="adm-trow-top">
                  {@render nameCell(t)}
                  {@render revokeButton(t)}
                </div>
                {@render scopeBadges(t.scopes)}
                <div class="adm-trow-meta">
                  <span>{@render lastUsed(t)}</span>
                  {@render expiry(t)}
                </div>
              </div>
            {/each}
          </div>
        {:else}
          <div class="adm-tokens" role="table" aria-label="令牌">
            <div class="adm-grid head" role="row">
              <span role="columnheader">名称</span>
              <span role="columnheader">权限</span>
              <span role="columnheader">最近使用</span>
              <span role="columnheader">到期</span>
              <span role="columnheader" class="sr-only">操作</span>
            </div>
            {#each tokens as t (t.id)}
              <div class="adm-grid" class:adm-revoked={!!t.revokedAt} role="row">
                <span role="cell">{@render nameCell(t)}</span>
                <span role="cell">{@render scopeBadges(t.scopes)}</span>
                <span role="cell">{@render lastUsed(t)}</span>
                <span role="cell">{@render expiry(t)}</span>
                <span role="cell" class="end">{@render revokeButton(t)}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>
{#if creating}
  <TokenDialog onclose={() => (creating = false)} onfinished={() => void load()} />
{/if}
