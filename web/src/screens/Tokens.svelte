<script lang="ts">
  import { onMount } from 'svelte';
  import Button from '../components/Button.svelte';
  import ConfirmDialog from '../components/ConfirmDialog.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import PageHeader from '../components/PageHeader.svelte';
  import { api, errorMessage } from '../lib/api';
  import { formatAbsolute, formatDate, formatRelative } from '../lib/format';
  import { toast } from '../lib/toast.svelte';
  import type { Scope, Token } from '../lib/types';
  import TokenDialog from './TokenDialog.svelte';

  const SCOPE_LABEL: Record<Scope, string> = { upload: '上传', read: '只读', manage: '管理' };

  let tokens = $state<Token[]>([]);
  let phase = $state<'loading' | 'ready' | 'error'>('loading');
  let errorText = $state('');
  let creating = $state(false);
  let revoking = $state<Token | null>(null);
  let revokeOpen = $state(false);

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

  function askRevoke(t: Token) {
    revoking = t;
    revokeOpen = true;
  }
  async function revoke() {
    const t = revoking;
    if (!t) return;
    try {
      await api.revokeToken(t.id);
      toast(`已吊销「${t.name}」`);
      await load();
    } catch (e) {
      toast(errorMessage(e));
    }
  }

  function expiryOf(t: Token): { text: string; expired: boolean } {
    if (!t.expiresAt) return { text: '永不过期', expired: false };
    const expired = new Date(t.expiresAt).getTime() < Date.now();
    return { text: formatDate(t.expiresAt), expired };
  }
  // Prefixes are stored without the ellipsis; it marks "the rest is secret".
  const shownPrefix = (p: string) => (p.endsWith('…') ? p : `${p}…`);
</script>

<div class="page tokens">
  <PageHeader title="令牌" meta="给 PicGo、ShareX 和脚本使用的上传凭证" mono={false}>
    {#snippet actions()}
      <Button variant="primary" icon="plus" onclick={() => (creating = true)}>新建令牌</Button>
    {/snippet}
  </PageHeader>

  {#if phase === 'loading'}
    <p class="muted">加载中…</p>
  {:else if phase === 'error'}
    <EmptyState message={`加载失败：${errorText}`}>
      <Button onclick={() => void load()}>重试</Button>
    </EmptyState>
  {:else if tokens.length === 0}
    <EmptyState title="还没有令牌" message="新建一个令牌，就能用 PicGo、ShareX 或脚本上传照片。">
      <Button variant="primary" icon="plus" onclick={() => (creating = true)}>新建令牌</Button>
    </EmptyState>
  {:else}
    <ul class="rows" aria-label="令牌列表">
      {#each tokens as t (t.id)}
        {@const exp = expiryOf(t)}
        {@const revokedRow = !!t.revokedAt}
        <li class="row" class:revoked={revokedRow}>
          <div class="id">
            <p class="name-line">
              <span class="name">{t.name}</span>
              {#if revokedRow}<span class="stamp">已吊销</span>{/if}
            </p>
            <p class="prefix mono">{shownPrefix(t.prefix)}</p>
          </div>

          <div class="scopes">
            {#each t.scopes as s (s)}
              <span class="stamp" class:solid={s === 'manage' && !revokedRow}>{SCOPE_LABEL[s] ?? s}</span>
            {/each}
          </div>

          <dl class="meta">
            <div>
              <dt>创建</dt>
              <dd title={formatAbsolute(t.createdAt)}>{formatRelative(t.createdAt)}</dd>
            </div>
            <div>
              <dt>最近使用</dt>
              <dd title={t.lastUsedAt ? formatAbsolute(t.lastUsedAt) : undefined}>
                {t.lastUsedAt ? formatRelative(t.lastUsedAt) : '从未使用'}
              </dd>
            </div>
            <div>
              <dt>过期</dt>
              <dd class="exp" title={t.expiresAt ? formatAbsolute(t.expiresAt) : undefined}>
                {#if exp.expired && !revokedRow}<span class="stamp danger">已过期</span>{/if}
                {exp.text}
              </dd>
            </div>
          </dl>

          <div class="act">
            {#if !revokedRow}
              <Button variant="danger" size="sm" aria-label="吊销「{t.name}」" onclick={() => askRevoke(t)}>吊销</Button>
            {/if}
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<TokenDialog bind:open={creating} onfinished={() => void load()} />

<ConfirmDialog
  bind:open={revokeOpen}
  title={revoking ? `吊销「${revoking.name}」？` : '吊销令牌？'}
  message="吊销后，使用此令牌的客户端会立即失效，且无法恢复。"
  confirmLabel="吊销"
  danger
  onconfirm={revoke}
/>

<style>
  .rows {
    border-top: 1px solid var(--ink);
  }
  .row {
    display: grid;
    grid-template-columns: minmax(0, 1.3fr) 156px minmax(0, 1.1fr) 64px;
    grid-template-areas: 'id scopes meta act';
    align-items: center;
    column-gap: var(--s5);
    padding: 17px 0;
    border-bottom: 1px solid var(--line);
  }
  .id {
    grid-area: id;
    min-width: 0;
  }
  .name-line {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--s1) var(--s2);
  }
  .name {
    font-size: 15px;
    font-weight: 650;
    line-height: 1.35;
  }
  .prefix {
    margin-top: 3px;
    color: var(--ink-3);
    font-size: var(--t-12);
  }
  .scopes {
    grid-area: scopes;
    display: flex;
    flex-wrap: wrap;
    gap: var(--s1);
  }
  .meta {
    grid-area: meta;
    display: grid;
    grid-template-columns: 4.5em minmax(0, 1fr);
    column-gap: var(--s2);
    row-gap: 2px;
    color: var(--ink-3);
    font-size: var(--t-13);
    line-height: 1.5;
  }
  .meta div {
    display: contents;
  }
  .meta dd {
    min-width: 0;
  }
  .exp {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--s1) var(--s2);
  }
  .act {
    grid-area: act;
    justify-self: end;
  }

  .revoked {
    color: var(--ink-3);
  }
  .revoked .name {
    text-decoration: line-through;
    text-decoration-thickness: 1px;
  }
  .revoked .scopes .stamp {
    color: var(--ink-3);
  }

  @media (max-width: 760px) {
    .row {
      grid-template-columns: minmax(0, 1fr) auto;
      grid-template-areas:
        'id act'
        'scopes scopes'
        'meta meta';
      align-items: start;
      row-gap: var(--s2);
      padding: var(--s3) 0 14px;
    }
    .meta {
      display: flex;
      flex-wrap: wrap;
      gap: 2px var(--s4);
    }
    .meta div {
      display: flex;
      gap: var(--s1);
    }
    .scopes:empty {
      display: none;
    }
  }
</style>
