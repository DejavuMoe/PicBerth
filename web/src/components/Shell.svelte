<script lang="ts">
  import { tick, type Snippet } from 'svelte';
  import { health } from '../lib/health.svelte';
  import type { IconName } from '../lib/icons';
  import { phone, tablet } from '../lib/media.svelte';
  import { canLeave, router } from '../lib/router.svelte';
  import { logout, session } from '../lib/session.svelte';
  import { setTheme, theme, THEME_OPTIONS } from '../lib/theme.svelte';
  import { toast } from '../lib/toast.svelte';
  import { uploads } from '../lib/upload.svelte';
  import Brand from './Brand.svelte';
  import Icon from './Icon.svelte';
  import Menu from './Menu.svelte';
  import MenuItem from './MenuItem.svelte';
  import Seg from './Seg.svelte';
  import Sheet from './Sheet.svelte';

  let { children }: { children: Snippet } = $props();

  interface NavEntry {
    href: string;
    match: string[];
    label: string;
    /** Label under the icon in the phone tab bar. */
    short: string;
    icon: IconName;
    /** A gap separates one group from the next in the sidebar. */
    group: number;
    /** Phone: a tab of its own; the others live in the 更多 sheet. */
    tab: boolean;
  }
  const NAV: NavEntry[] = [
    { href: '/library', match: ['/', '/library'], label: '图片', short: '图片', icon: 'images', group: 0, tab: true },
    { href: '/trash', match: ['/trash'], label: '回收站', short: '回收站', icon: 'trash', group: 0, tab: true },
    { href: '/tokens', match: ['/tokens'], label: '令牌', short: '令牌', icon: 'key', group: 1, tab: true },
    { href: '/settings', match: ['/settings'], label: '设置', short: '设置', icon: 'gear', group: 2, tab: false },
    { href: '/status', match: ['/status'], label: '状态', short: '状态', icon: 'activity', group: 2, tab: false },
  ];
  const TABS = NAV.filter((n) => n.tab);
  const MORE = NAV.filter((n) => !n.tab);

  const current = $derived(NAV.find((n) => n.match.includes(router.path)));
  const warnings = $derived(health.status?.warnings.length ?? 0);
  const danger = $derived(health.lamp === 'danger');
  // Anything the status page should be opened for: warnings, or a failed consistency check.
  const attention = $derived(warnings > 0 || danger);
  const attentionLabel = $derived(warnings > 0 ? `${warnings} 条警告` : health.label);
  const name = $derived(session.user?.name ?? '');

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
          {#if i > 0 && NAV[i - 1]?.group !== n.group}<div class="nav-gap"></div>{/if}
          <a class="nav-item" href={n.href} title={n.label} aria-current={n.match.includes(router.path) ? 'page' : undefined}>
            <Icon name={n.icon} />
            <span class="nav-label">{n.label}</span>
            {#if n.href === '/status' && attention}
              <span class="warn-dot" class:danger role="img" aria-label={attentionLabel}></span>
            {/if}
          </a>
        {/each}
      </nav>
      <Menu up width={200} wrapClass="account-wrap" label="账户">
        {#snippet trigger(open, toggle)}
          <button type="button" class="account" aria-label="账户" title={name} aria-haspopup="menu" aria-expanded={open} onclick={toggle}>
            <span class="avatar">{name.slice(0, 1).toUpperCase()}</span>
            <span class="account-name">{name}</span>
          </button>
        {/snippet}
        {#snippet children(close)}
          {#each THEME_OPTIONS as t (t.key)}
            <MenuItem selected={theme.choice === t.key} onclick={() => (setTheme(t.key), close())}>{t.label}</MenuItem>
          {/each}
          <div class="menu-sep"></div>
          <MenuItem icon="logout" onclick={() => (close(), signOut())}>退出登录</MenuItem>
        {/snippet}
      </Menu>
    </aside>
  {/if}

  <div class="panel">
    <main id="main" class="panel-main" tabindex="-1" bind:this={main}>
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
      <span class="tab-icon">
        <Icon name="more" size={20} />
        {#if attention}<span class="warn-dot" class:danger aria-hidden="true"></span>{/if}
      </span>
      <span>更多</span>
    </button>
  </nav>
{/if}

{#if moreOpen && phone.matches}
  <Sheet label="更多" onclose={() => (moreOpen = false)}>
    {#snippet children(close)}
      <div class="sheet-body more-body">
        {#each MORE as n (n.href)}
          <a class="nav-item" href={n.href} aria-current={n.match.includes(router.path) ? 'page' : undefined} onclick={close}>
            <Icon name={n.icon} />
            <span class="nav-label">{n.label}</span>
            {#if n.href === '/status' && attention}
              <span class="warn-dot" class:danger role="img" aria-label={attentionLabel}></span>
            {/if}
          </a>
        {/each}
        <Seg fill label="主题" value={theme.choice} onchange={setTheme} options={THEME_OPTIONS} />
        <button type="button" class="nav-item" onclick={() => (close(), signOut())}>
          <Icon name="logout" />
          <span class="nav-label">退出登录</span>
        </button>
      </div>
    {/snippet}
  </Sheet>
{/if}

<div class="sr-only" role="status" aria-live="polite">{uploads.announcement}</div>
