<script lang="ts">
  import { tick, type Snippet } from 'svelte';
  import { facets } from '../lib/facets.svelte';
  import { formatBytes } from '../lib/format';
  import { health } from '../lib/health.svelte';
  import type { IconName } from '../lib/icons';
  import { phone, tablet } from '../lib/media.svelte';
  import { canLeave, router } from '../lib/router.svelte';
  import { logout, session } from '../lib/session.svelte';
  import { theme, toggleTheme } from '../lib/theme.svelte';
  import { toast } from '../lib/toast.svelte';
  import { uploads } from '../lib/upload.svelte';
  import Brand from './Brand.svelte';
  import Icon from './Icon.svelte';
  import Menu from './Menu.svelte';
  import MenuItem from './MenuItem.svelte';
  import Sheet from './Sheet.svelte';
  import Switch from './Switch.svelte';

  let { children }: { children: Snippet } = $props();

  interface NavEntry {
    href: string;
    match: string[];
    label: string;
    /** Label under the icon in the phone tab bar. */
    short: string;
    icon: IconName;
    group: string;
    /** Phone: a tab of its own; the others live in the 更多 sheet. */
    tab: boolean;
    count?: () => number | undefined;
  }
  const NAV: NavEntry[] = [
    { href: '/library', match: ['/', '/library'], label: '全部图片', short: '图片', icon: 'images', group: '图片', tab: true, count: () => facets.active?.total },
    { href: '/trash', match: ['/trash'], label: '回收站', short: '回收站', icon: 'trash', group: '图片', tab: true, count: () => facets.trashed?.total },
    { href: '/tokens', match: ['/tokens'], label: '令牌', short: '令牌', icon: 'key', group: '接入', tab: true },
    { href: '/settings', match: ['/settings'], label: '设置', short: '设置', icon: 'gear', group: '系统', tab: false },
    { href: '/status', match: ['/status'], label: '状态', short: '状态', icon: 'activity', group: '系统', tab: false },
  ];
  const TABS = NAV.filter((n) => n.tab);
  const MORE = NAV.filter((n) => !n.tab);

  const current = $derived(NAV.find((n) => n.match.includes(router.path)));
  const warnings = $derived(health.status?.warnings.length ?? 0);
  const danger = $derived(health.lamp === 'danger');
  // Anything the status page should be opened for: warnings, or a failed consistency check.
  const attention = $derived(warnings > 0 || danger);
  const dark = $derived(theme.current === 'dark');
  const name = $derived(session.user?.name ?? '');

  const disk = $derived(health.status?.disk ?? null);
  const diskUsed = $derived(disk && disk.totalBytes > 0 ? Math.round((1 - disk.freeBytes / disk.totalBytes) * 100) : 0);

  let moreOpen = $state(false);
  $effect(() => {
    if (!phone.matches) moreOpen = false;
  });
  const moreOn = $derived(moreOpen || (current !== undefined && !current.tab));

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

<a class="skip" href="#main" onclick={(e) => { e.preventDefault(); main?.focus(); }}>跳到内容</a>

<div class="app" class:rail={tablet.matches} class:mobile={phone.matches}>
  {#if !phone.matches}
    <aside class="sidenav" aria-label="主导航">
      <Brand />
      <nav>
        {#each NAV as n, i (n.href)}
          {#if i === 0 || NAV[i - 1]?.group !== n.group}<div class="nav-group">{n.group}</div>{/if}
          {@const count = n.count?.()}
          {@const flagged = n.href === '/status' && attention}
          <a class="nav-item" href={n.href} title={n.label} aria-current={n.match.includes(router.path) ? 'page' : undefined}>
            <span class="nav-icon">
              <Icon name={n.icon} />
              {#if flagged}<span class="nav-dot" class:danger aria-hidden="true"></span>{/if}
            </span>
            <span class="nav-label">{n.label}</span>
            {#if count !== undefined}<span class="count num">{count}</span>{/if}
            {#if flagged}
              <span class="count" aria-label={warnings > 0 ? `${warnings} 条警告` : health.label}>
                {#if warnings > 0}
                  <span class="badge badge-xs" class:badge-warning={!danger} class:badge-danger={danger}>{warnings}</span>
                {:else}
                  <span class="dot" style:color="var(--danger)"></span>
                {/if}
              </span>
            {/if}
          </a>
        {/each}
      </nav>
      {#if !tablet.matches}
        <div class="usage">
          <div class="usage-row">
            <span>磁盘剩余</span>
            <span class="num">{disk ? formatBytes(disk.freeBytes) : '—'}</span>
          </div>
          <div class="progress" role="progressbar" aria-label="磁盘已用" aria-valuemin={0} aria-valuemax={100} aria-valuenow={diskUsed}>
            <span style:width="{diskUsed}%"></span>
          </div>
        </div>
      {/if}
    </aside>
  {/if}

  <div class="main-col">
    <header class="topbar">
      {#if phone.matches}
        <Brand />
      {:else}
        <nav class="crumbs" aria-label="位置">
          <span>PicBerth</span>
          <Icon name="chevron-right" size={12} />
          {#if current && current.group !== '图片'}
            <span>{current.group}</span>
            <Icon name="chevron-right" size={12} />
          {/if}
          <strong>{current?.label ?? '页面不存在'}</strong>
        </nav>
      {/if}
      <span class="grow"></span>
      {#if !phone.matches}
        <button
          type="button"
          class="btn btn-ghost btn-icon"
          aria-label={dark ? '切换到浅色' : '切换到深色'}
          title={dark ? '切换到浅色' : '切换到深色'}
          onclick={toggleTheme}
        >
          <Icon name={dark ? 'sun' : 'moon'} />
        </button>
      {/if}
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

{#if phone.matches}
  <nav class="tabbar" aria-label="主导航">
    {#each TABS as n (n.href)}
      <a class="tab" href={n.href} aria-current={n.match.includes(router.path) && !moreOpen ? 'page' : undefined}>
        <Icon name={n.icon} size={20} />
        <span>{n.short}</span>
      </a>
    {/each}
    <button type="button" class="tab" aria-current={moreOn ? 'page' : undefined} aria-expanded={moreOpen} onclick={() => (moreOpen = true)}>
      <span class="nav-icon">
        <Icon name="more" size={20} />
        {#if attention}<span class="nav-dot" class:danger aria-hidden="true"></span>{/if}
      </span>
      <span>更多</span>
    </button>
  </nav>
{/if}

{#if moreOpen && phone.matches}
  <Sheet label="更多" onclose={() => (moreOpen = false)}>
    {#snippet children(close)}
      <div class="sheet-head">
        <h2>更多</h2>
      </div>
      <div class="sheet-body more-body">
        {#each MORE as n (n.href)}
          <a class="nav-item" href={n.href} aria-current={n.match.includes(router.path) ? 'page' : undefined} onclick={close}>
            <span class="nav-icon"><Icon name={n.icon} /></span>
            <span class="nav-label">{n.label}</span>
            {#if n.href === '/status' && attention}
              <span class="count">
                {#if warnings > 0}
                  <span class="badge" class:badge-warning={!danger} class:badge-danger={danger}>{warnings} 条警告</span>
                {:else}
                  <span class="badge badge-danger">{health.label}</span>
                {/if}
              </span>
            {/if}
            <span class="count"><Icon name="chevron-right" /></span>
          </a>
        {/each}
        <div class="menu-sep"></div>
        <div class="setting-row more-row">
          <span class="label">深色模式</span>
          <Switch checked={dark} label="深色模式" onchange={toggleTheme} />
        </div>
      </div>
    {/snippet}
  </Sheet>
{/if}

<div class="sr-only" role="status" aria-live="polite">{uploads.announcement}</div>
