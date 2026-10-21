<script lang="ts">
  import { tick, type Snippet } from 'svelte';
  import { facets } from '../lib/facets.svelte';
  import { formatBytes } from '../lib/format';
  import { health } from '../lib/health.svelte';
  import type { IconName } from '../lib/icons';
  import { canLeave, router } from '../lib/router.svelte';
  import { logout, session } from '../lib/session.svelte';
  import { theme, toggleTheme } from '../lib/theme.svelte';
  import { toast } from '../lib/toast.svelte';
  import { uploads } from '../lib/upload.svelte';
  import Brand from './Brand.svelte';
  import Drawer from './Drawer.svelte';
  import Icon from './Icon.svelte';
  import Menu from './Menu.svelte';
  import MenuItem from './MenuItem.svelte';

  let { children }: { children: Snippet } = $props();

  interface NavEntry {
    href: string;
    match: string[];
    label: string;
    icon: IconName;
    group: string;
    count?: () => number | undefined;
  }
  const NAV: NavEntry[] = [
    { href: '/library', match: ['/', '/library'], label: '全部图片', icon: 'images', group: '图片', count: () => facets.active?.total },
    { href: '/trash', match: ['/trash'], label: '回收站', icon: 'trash', group: '图片', count: () => facets.trashed?.total },
    { href: '/tokens', match: ['/tokens'], label: '令牌', icon: 'key', group: '接入' },
    { href: '/settings', match: ['/settings'], label: '设置', icon: 'gear', group: '系统' },
    { href: '/status', match: ['/status'], label: '状态', icon: 'activity', group: '系统' },
  ];

  const current = $derived(NAV.find((n) => n.match.includes(router.path)));
  const warnings = $derived(health.status?.warnings.length ?? 0);
  const dark = $derived(theme.current === 'dark');
  const name = $derived(session.user?.name ?? '');

  const imageBytes = $derived(facets.active ? facets.active.bytes + (facets.trashed?.bytes ?? 0) : null);
  const disk = $derived(health.status?.disk ?? null);
  const diskUsed = $derived(disk && disk.totalBytes > 0 ? Math.round((1 - disk.freeBytes / disk.totalBytes) * 100) : 0);

  let navOpen = $state(false);
  $effect(() => {
    router.path;
    navOpen = false;
  });

  // Move focus to the content after in-app navigation so keyboard and screen reader users start there.
  let main = $state<HTMLElement | null>(null);
  let firstPath = true;
  $effect(() => {
    router.path;
    if (firstPath) {
      firstPath = false;
      return;
    }
    void tick().then(() => main?.focus({ preventScroll: true }));
  });

  async function signOut() {
    // Leave guards (unsaved settings) apply before the session is dropped.
    if (!(await canLeave('/login'))) return;
    try {
      await logout();
      toast('已退出登录');
    } catch {
      toast('退出失败，请重试');
    }
  }
</script>

{#snippet navList()}
  {#each NAV as n, i (n.href)}
    {#if i === 0 || NAV[i - 1]?.group !== n.group}<div class="nav-group">{n.group}</div>{/if}
    {@const count = n.count?.()}
    <a class="nav-item" href={n.href} aria-current={n.match.includes(router.path) ? 'page' : undefined}>
      <Icon name={n.icon} />
      {n.label}
      {#if count !== undefined}<span class="count">{count}</span>{/if}
      {#if n.href === '/status' && (warnings > 0 || health.lamp === 'danger')}
        <span class="count" aria-label={warnings > 0 ? `${warnings} 条警告` : health.label}>
          <span class="dot" style:color={health.lamp === 'danger' ? 'var(--danger)' : 'var(--warning)'}></span>
        </span>
      {/if}
    </a>
  {/each}
{/snippet}

{#snippet usage()}
  <div class="usage">
    <div class="usage-row">
      <span>图片占用</span>
      <span class="num">{imageBytes === null ? '—' : formatBytes(imageBytes)}</span>
    </div>
    <div class="progress" role="progressbar" aria-label="磁盘已用" aria-valuemin={0} aria-valuemax={100} aria-valuenow={diskUsed}>
      <span style:width="{diskUsed}%"></span>
    </div>
    <div class="usage-row">
      <span>磁盘剩余</span>
      <span class="num">{disk ? formatBytes(disk.freeBytes) : '—'}</span>
    </div>
  </div>
{/snippet}

<a class="skip" href="#main" onclick={(e) => { e.preventDefault(); main?.focus(); }}>跳到内容</a>

<div class="shell">
  <aside class="sidenav" aria-label="主导航">
    <Brand />
    <nav>{@render navList()}</nav>
    {@render usage()}
  </aside>

  <div class="main-col">
    <header class="topbar">
      <button type="button" class="btn btn-ghost btn-icon only-mobile" aria-label="打开导航" onclick={() => (navOpen = true)}>
        <Icon name="menu" />
      </button>
      <nav class="crumbs" aria-label="位置">
        <span class="hide-sm">PicBerth</span>
        <span class="hide-sm">/</span>
        {#if current && current.group !== '图片'}
          <span class="hide-sm">{current.group}</span>
          <span class="hide-sm">/</span>
        {/if}
        <strong>{current?.label ?? '页面不存在'}</strong>
      </nav>
      <span class="grow"></span>
      <button
        type="button"
        class="btn btn-ghost btn-icon"
        aria-label={dark ? '切换到浅色' : '切换到深色'}
        title={dark ? '切换到浅色' : '切换到深色'}
        onclick={toggleTheme}
      >
        <Icon name={dark ? 'sun' : 'moon'} />
      </button>
      <Menu align="right" width={180}>
        {#snippet trigger(open, toggle)}
          <button type="button" class="btn btn-ghost btn-icon" aria-label="账户" aria-expanded={open} onclick={toggle}>
            <span class="avatar">{name.slice(0, 1).toUpperCase()}</span>
          </button>
        {/snippet}
        {#snippet children(close)}
          <div class="menu-label">{name}</div>
          <div class="menu-sep"></div>
          <MenuItem icon="logout" onclick={() => (close(), signOut())}>退出登录</MenuItem>
        {/snippet}
      </Menu>
    </header>
    <main id="main" class="main" tabindex="-1" bind:this={main}>
      {#key router.path === '/library' ? '/' : router.path}
        {@render children()}
      {/key}
    </main>
  </div>
</div>

{#if navOpen}
  <Drawer side="left" width="272px" label="导航" onclose={() => (navOpen = false)}>
    <div class="sidenav in-drawer">
      <Brand />
      <nav>{@render navList()}</nav>
      {@render usage()}
    </div>
  </Drawer>
{/if}

<div class="sr-only" role="status" aria-live="polite">{uploads.announcement}</div>
