<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from '../components/Icon.svelte';
  import Modal from '../components/Modal.svelte';
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
  let revoking = $state<Token | null>(null);

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

  async function revoke() {
    const t = revoking;
    revoking = null;
    if (!t) return;
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

{#snippet newButton()}
  <button type="button" class="btn btn-primary" onclick={() => (creating = true)}>
    <Icon name="plus" />
    新建令牌
  </button>
{/snippet}

<div class="page" style="--page-w: 1040px">
  <div class="page-head">
    <div>
      <h1>令牌</h1>
      <p>用于 PicGo、ShareX 和脚本上传</p>
    </div>
    {#if phase === 'ready' && tokens.length > 0}<div class="page-actions">{@render newButton()}</div>{/if}
  </div>

  {#if phase === 'loading'}
    <p class="subtle">加载中…</p>
  {:else if phase === 'error'}
    <div class="alert danger" role="alert">
      <Icon name="alert" />
      <div class="grow">加载失败：{errorText}</div>
      <button type="button" class="btn btn-secondary btn-sm" onclick={() => void load()}>重试</button>
    </div>
  {:else if tokens.length === 0}
    <div class="empty">
      <strong>还没有令牌</strong>
      <p>新建令牌后，可以在 PicGo、ShareX 或脚本中上传图片。</p>
      <div class="actions">{@render newButton()}</div>
    </div>
  {:else if phone.matches}
    <div class="adm-tcards">
      {#each tokens as t (t.id)}
        <div class="card adm-tcard" class:adm-revoked={!!t.revokedAt}>
          <div class="adm-tcard-top">
            <div class="cell-name">
              <span class="name truncate">{t.name}</span>
              <span class="sub mono">{shownPrefix(t.prefix)}</span>
            </div>
            {#if t.revokedAt}
              <span class="badge">已吊销</span>
            {:else}
              <button type="button" class="btn btn-danger btn-sm" aria-label={`吊销「${t.name}」`} onclick={() => (revoking = t)}>吊销</button>
            {/if}
          </div>
          {@render scopeBadges(t.scopes)}
          <div class="adm-kv">
            <span>最近使用</span>
            <span>{@render lastUsed(t)}</span>
          </div>
          {#if !t.revokedAt}
            <div class="adm-kv">
              <span>过期</span>
              <span>{@render expiry(t)}</span>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {:else}
    <div class="table-wrap">
      <table class="table adm-table">
        <thead>
          <tr>
            <th>名称</th>
            <th>权限</th>
            <th>最近使用</th>
            <th>过期</th>
            <th class="c-actions"><span class="sr-only">操作</span></th>
          </tr>
        </thead>
        <tbody>
          {#each tokens as t (t.id)}
            <tr class:adm-revoked={!!t.revokedAt}>
              <td>
                <div class="cell-name">
                  <span class="name truncate">{t.name}</span>
                  <span class="sub mono">{shownPrefix(t.prefix)}</span>
                </div>
              </td>
              <td>{@render scopeBadges(t.scopes)}</td>
              <td>{@render lastUsed(t)}</td>
              <td>{@render expiry(t)}</td>
              <td class="c-actions">
                {#if !t.revokedAt}
                  <button type="button" class="btn btn-danger btn-sm" aria-label={`吊销「${t.name}」`} onclick={() => (revoking = t)}>吊销</button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if creating}
  <TokenDialog onclose={() => (creating = false)} onfinished={() => void load()} />
{/if}

{#if revoking}
  <Modal title={`吊销令牌「${revoking.name}」？`} width="440px" onclose={() => (revoking = null)}>
    <p>使用此令牌的工具将无法继续上传。</p>
    {#snippet footer()}
      <button type="button" class="btn btn-secondary" data-autofocus onclick={() => (revoking = null)}>取消</button>
      <button type="button" class="btn btn-danger-solid" onclick={revoke}>吊销</button>
    {/snippet}
  </Modal>
{/if}
