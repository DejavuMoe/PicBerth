<script lang="ts">
  import { onDestroy, onMount, tick, type Snippet } from 'svelte';
  import { facets } from '../lib/facets.svelte';
  import { health } from '../lib/health.svelte';
  import type { IconName } from '../lib/icons';
  import { phone, wideNav } from '../lib/media.svelte';
  import { MOD, openPalette, palette } from '../lib/palette.svelte';
  import { canLeave, isLibraryPath, router } from '../lib/router.svelte';
  import { logout, session } from '../lib/session.svelte';
  import { setTheme, theme, THEME_OPTIONS } from '../lib/theme.svelte';
  import { toast } from '../lib/toast.svelte';
  import { tree } from '../lib/tree.svelte';
  import { uploads } from '../lib/upload.svelte';
  import FolderPanel from './library/FolderPanel.svelte';
  import Icon from './Icon.svelte';
  import Menu from './Menu.svelte';
  import MenuItem from './MenuItem.svelte';
  import UploadDock from './UploadDock.svelte';

  let { children }: { children: Snippet } = $props();

  interface NavEntry {
    href: string;
    match: string[];
    label: string;
    icon: IconName;
    /** A gap separates one group from the next in the sidebar. */
    group: number;
  }
  const NAV: NavEntry[] = [
    { href: '/library', match: ['/', '/library'], label: '图片', icon: 'image', group: 0 },
    { href: '/trash', match: ['/trash'], label: '回收站', icon: 'trash', group: 0 },
    { href: '/tokens', match: ['/tokens'], label: '令牌', icon: 'key', group: 1 },
    { href: '/settings', match: ['/settings'], label: '设置', icon: 'sliders', group: 1 },
    { href: '/status', match: ['/status'], label: '状态', icon: 'pulse', group: 1 },
  ];

  const isImages = $derived(router.path === '/' || isLibraryPath(router.path));
  const isCurrent = (n: NavEntry) => (n.href === '/library' ? isImages : n.match.includes(router.path));
  const screenKey = $derived(isImages ? 'library' : router.path);
  const warnings = $derived(health.status?.warnings.length ?? 0);
  const danger = $derived(health.lamp === 'danger');
  const trashCount = $derived(facets.trashed?.total ?? 0);
  const name = $derived(session.user?.name ?? '');

  // The folder tree is part of the navigation on every screen.
  onMount(() => void tree.ensure());
  onDestroy(() => tree.reset());

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

{#snippet navLink(n: NavEntry)}
  <a class="nav-item" href={n.href} title={n.label} aria-current={isCurrent(n) ? 'page' : undefined}>
    <Icon name={n.icon} />
    <span class="label">{n.label}</span>
    {#if n.href === '/trash' && trashCount > 0}<span class="count">{trashCount}</span>{/if}
    {#if n.href === '/status' && (warnings > 0 || danger)}
      <span class="dot {danger ? 'dot-danger' : 'dot-warning'}" role="img" aria-label={warnings > 0 ? `${warnings} 条警告` : health.label}></span>
    {/if}
  </a>
{/snippet}

<a class="skip" href="#main" onclick={(e) => { e.preventDefault(); main?.focus(); }}>跳到内容</a>

<div class="shell">
  <nav class="side" aria-label="主导航">
    <div class="brand-row">
      <span class="brand">
        <span class="brand-mark" aria-hidden="true"></span>
        <span class="brand-word">PicBerth</span>
      </span>
      <button
        type="button"
        class="cmdk-btn"
        aria-label="命令面板"
        title="命令面板"
        aria-haspopup="dialog"
        aria-expanded={palette.open}
        onclick={openPalette}
      >
        {MOD} K
      </button>
    </div>
    {#if wideNav.matches}
      <div class="side-scroll">
        <FolderPanel />
      </div>
    {:else}
      <div class="side-nav">{@render navLink(NAV[0]!)}</div>
    {/if}
    <div class="side-nav">{@render navLink(NAV[1]!)}</div>
    <div class="side-sep"></div>
    <div class="side-nav">
      {#each NAV.filter((n) => n.group === 1) as n (n.href)}
        {@render navLink(n)}
      {/each}
    </div>
    <div class="side-fill"></div>
    <div class="side-nav side-foot">
      <Menu align="left" width={200} label="账户">
        {#snippet trigger(open, toggle)}
          <button type="button" class="account" aria-label={name} title={name} aria-haspopup="menu" aria-expanded={open} onclick={toggle}>
            <span class="avatar">{name.slice(0, 1).toUpperCase()}</span>
            <span class="label">{name}</span>
          </button>
        {/snippet}
        {#snippet children(close)}
          <div class="menu-group">主题</div>
          {#each THEME_OPTIONS as t (t.key)}
            <MenuItem checked={theme.choice === t.key} onclick={() => (setTheme(t.key), close())}>{t.label}</MenuItem>
          {/each}
          <div class="menu-sep"></div>
          <MenuItem icon="logout" onclick={() => (close(), signOut())}>退出</MenuItem>
        {/snippet}
      </Menu>
    </div>
  </nav>

  <main class="main">
    <div class="panel" id="main" tabindex="-1" bind:this={main}>
      {#key screenKey}
        {@render children()}
      {/key}
    </div>
  </main>

  {#if phone.matches}
    <nav class="tabbar" aria-label="主导航">
      {#each NAV as n (n.href)}
        <a class="tab" href={n.href} aria-current={isCurrent(n) ? 'page' : undefined}>
          <Icon name={n.icon} />
          {n.label}
          {#if n.href === '/status' && (warnings > 0 || danger)}
            <span class="dot {danger ? 'dot-danger' : 'dot-warning'}" role="img" aria-label={warnings > 0 ? `${warnings} 条警告` : health.label}></span>
          {/if}
        </a>
      {/each}
    </nav>
  {/if}
</div>

<UploadDock />

<div class="sr-only" role="status" aria-live="polite">{uploads.announcement}</div>
