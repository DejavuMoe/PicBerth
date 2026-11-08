<script lang="ts">
  import { tick, type Snippet } from 'svelte';
  import { facets } from '../lib/facets.svelte';
  import { health } from '../lib/health.svelte';
  import type { IconName } from '../lib/icons';
  import { phone } from '../lib/media.svelte';
  import { canLeave, router } from '../lib/router.svelte';
  import { logout, session } from '../lib/session.svelte';
  import { setTheme, theme, THEME_OPTIONS } from '../lib/theme.svelte';
  import { toast } from '../lib/toast.svelte';
  import { uploads } from '../lib/upload.svelte';
  import Icon from './Icon.svelte';
  import Menu from './Menu.svelte';
  import MenuItem from './MenuItem.svelte';

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

  const current = $derived(NAV.find((n) => n.match.includes(router.path)));
  const warnings = $derived(health.status?.warnings.length ?? 0);
  const danger = $derived(health.lamp === 'danger');
  const trashCount = $derived(facets.trashed?.total ?? 0);
  const name = $derived(session.user?.name ?? '');

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

<div class="shell">
  <nav class="side" aria-label="主导航">
    <div class="brand">
      <span class="brand-mark" aria-hidden="true"></span>
      <span class="brand-word">PicBerth</span>
    </div>
    {#each [0, 1] as g (g)}
      <div class="nav">
        {#each NAV.filter((n) => n.group === g) as n (n.href)}
          <a class="nav-item" href={n.href} title={n.label} aria-current={n.match.includes(router.path) ? 'page' : undefined}>
            <Icon name={n.icon} />
            <span class="label">{n.label}</span>
            {#if n.href === '/trash' && trashCount > 0}<span class="count">{trashCount}</span>{/if}
            {#if n.href === '/status' && (warnings > 0 || danger)}
              <span class="dot {danger ? 'dot-danger' : 'dot-warning'}" role="img" aria-label={warnings > 0 ? `${warnings} 条警告` : health.label}></span>
            {/if}
          </a>
        {/each}
      </div>
    {/each}
    <div class="side-foot">
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
      {#key router.path === '/library' ? '/' : router.path}
        {@render children()}
      {/key}
    </div>
  </main>

  {#if phone.matches}
    <nav class="tabbar" aria-label="主导航">
      {#each NAV as n (n.href)}
        <a class="tab" href={n.href} aria-current={n.match.includes(router.path) ? 'page' : undefined}>
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

<div class="sr-only" role="status" aria-live="polite">{uploads.announcement}</div>
