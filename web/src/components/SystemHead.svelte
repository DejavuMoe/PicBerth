<script lang="ts">
  import type { Snippet } from 'svelte';
  import { phone } from '../lib/media.svelte';
  import { router } from '../lib/router.svelte';

  // 系统 groups 状态, 令牌 and 设置; each keeps its own route, so links (and leave guards) work as before.
  let { children }: { children?: Snippet } = $props();

  const TABS = [
    { href: '/status', label: '状态' },
    { href: '/tokens', label: '令牌' },
    { href: '/settings', label: '设置' },
  ];
</script>

{#snippet tabs(block: boolean)}
  <nav class="seg sys-tabs" class:seg-block={block} aria-label="系统">
    {#each TABS as t (t.href)}
      <a href={t.href} aria-current={router.path === t.href ? 'page' : undefined}>{t.label}</a>
    {/each}
  </nav>
{/snippet}

<div class="head">
  <div class="head-title sys-title">
    <h1>系统</h1>
    {#if !phone.matches}{@render tabs(false)}{/if}
  </div>
  {@render children?.()}
</div>
{#if phone.matches}
  <div class="subbar">{@render tabs(true)}</div>
{/if}
