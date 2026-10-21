<script lang="ts">
  import { health } from '../lib/health.svelte';
  import { openPicker } from '../lib/picker';
  import { canLeave, router } from '../lib/router.svelte';
  import { logout, session } from '../lib/session.svelte';
  import { theme, toggleTheme } from '../lib/theme.svelte';
  import { toast } from '../lib/toast.svelte';
  import { busy, uploads } from '../lib/upload.svelte';
  import type { IconName } from '../lib/icons';
  import Grease from './Grease.svelte';
  import Icon from './Icon.svelte';
  import IconButton from './IconButton.svelte';
  import Logo from './Logo.svelte';

  const NAV: { href: string; label: string; icon: IconName; match: string[] }[] = [
    { href: '/', label: '底片', icon: 'film', match: ['/', '/library'] },
    { href: '/trash', label: '回收站', icon: 'trash', match: ['/trash'] },
    { href: '/tokens', label: '令牌', icon: 'key', match: ['/tokens'] },
    { href: '/settings', label: '设置', icon: 'gear', match: ['/settings'] },
    { href: '/status', label: '状态', icon: 'pulse', match: ['/status'] },
  ];
  // The mobile tab bar keeps four destinations around the shutter; status lives on the header lamp.
  const TABS = NAV.filter((n) => n.href !== '/status');

  const developing = $derived(uploads.items.filter((i) => i.status === 'queued' || i.status === 'uploading' || i.status === 'processing').length);
  const themeLabel = $derived(theme.current === 'dark' ? '切换到亮台（浅色）' : '切换到暗房（深色）');

  async function signOut() {
    // Leave guards (unsaved settings) apply before the session is dropped.
    if (!(await canLeave('/login'))) return;
    try {
      await logout();
    } catch {
      toast('退出失败，请重试');
    }
  }
</script>

<header class="top">
  <a class="brand" href="/" aria-label="PicBerth 底片"><Logo /></a>

  <nav class="nav" aria-label="主导航">
    {#each NAV as item (item.href)}
      {@const current = item.match.includes(router.path)}
      <a href={item.href} aria-current={current ? 'page' : undefined}>
        <span class="lbl">
          {item.label}
          {#if current}
            {#key router.path}<Grease shape="line" seed={item.href} />{/key}
          {/if}
        </span>
      </a>
    {/each}
  </nav>

  <div class="tools">
    <a class="lamp-link" href="/status" title="状态：{health.label}" aria-label="状态：{health.label}">
      <span class="lamp {health.lamp}"></span>
    </a>
    <IconButton icon="lamp" label={themeLabel} onclick={toggleTheme} />
    {#if session.user}
      <span class="user" title="当前用户">{session.user.name}</span>
    {/if}
    <IconButton icon="logout" label="退出登录" onclick={signOut} />
    <button type="button" class="develop" onclick={openPicker} title="选择照片上传（U）">
      <Icon name="plus" />
      <span>冲洗照片</span>
      {#if developing > 0}<span class="badge edge" aria-label="{developing} 张冲洗中">{developing}</span>{/if}
    </button>
  </div>
</header>

<nav class="tabbar" aria-label="主导航（移动）">
  {#each TABS.slice(0, 2) as item (item.href)}
    {@const current = item.match.includes(router.path)}
    <a href={item.href} aria-current={current ? 'page' : undefined}>
      <Icon name={item.icon} size={20} />
      <span>{item.label}</span>
    </a>
  {/each}
  <button type="button" class="shutter" onclick={openPicker} aria-label="选择照片上传" class:busy={busy()}>
    <span class="ring"><span class="disc"></span></span>
  </button>
  {#each TABS.slice(2) as item (item.href)}
    {@const current = item.match.includes(router.path)}
    <a href={item.href} aria-current={current ? 'page' : undefined}>
      <Icon name={item.icon} size={20} />
      <span>{item.label}</span>
    </a>
  {/each}
</nav>

<style>
  .top {
    position: sticky;
    top: 0;
    z-index: 30;
    display: flex;
    align-items: center;
    gap: var(--s6);
    height: var(--bar-h);
    padding: 0 var(--gutter);
    background: color-mix(in srgb, var(--bg) 94%, transparent);
    border-bottom: 1px solid var(--line);
    transition: background-color 240ms var(--ease);
  }
  .brand {
    display: inline-flex;
    align-items: center;
    flex: none;
    height: 100%;
    color: var(--ink);
    text-decoration: none;
  }
  .nav {
    display: flex;
    align-items: stretch;
    gap: var(--s1);
    height: 100%;
    margin-left: calc(-1 * var(--s3));
  }
  .nav a {
    display: inline-flex;
    align-items: center;
    padding: 0 var(--s3);
    color: var(--ink-2);
    font-weight: 520;
    text-decoration: none;
    white-space: nowrap;
    transition: color var(--dur) var(--ease);
  }
  .nav a:hover {
    color: var(--ink);
  }
  .nav a[aria-current='page'] {
    color: var(--ink);
    font-weight: 650;
  }
  .lbl {
    position: relative;
  }

  .tools {
    display: flex;
    align-items: center;
    gap: 2px;
    margin-left: auto;
  }
  .lamp-link {
    display: grid;
    place-items: center;
    width: var(--control-h);
    height: var(--control-h);
    border-radius: var(--r-ctl);
  }
  .lamp-link:hover {
    background: var(--surface-2);
  }
  .user {
    max-width: 12ch;
    overflow: hidden;
    margin: 0 var(--s1) 0 var(--s2);
    color: var(--ink-3);
    font-size: var(--t-13);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .develop {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 7px;
    height: var(--control-h);
    margin-left: var(--s3);
    padding: 0 16px 0 12px;
    border-radius: var(--r-ctl);
    background: var(--ink);
    color: var(--bg);
    font-size: 13.5px;
    font-weight: 650;
    white-space: nowrap;
    transition: background-color var(--dur) var(--ease);
  }
  .develop:hover {
    background: color-mix(in srgb, var(--ink) 84%, var(--bg));
  }
  .badge {
    display: inline-grid;
    place-items: center;
    min-width: 18px;
    height: 18px;
    padding: 0 4px;
    border-radius: 2px;
    background: var(--edge);
    color: var(--film);
    font-weight: 700;
  }

  .tabbar {
    display: none;
  }

  @media (max-width: 1040px) {
    .user {
      display: none;
    }
    .top {
      gap: var(--s4);
    }
  }

  @media (max-width: 760px) {
    .nav,
    .develop {
      display: none;
    }
    .tabbar {
      position: fixed;
      left: 0;
      right: 0;
      bottom: 0;
      z-index: 30;
      display: grid;
      grid-template-columns: repeat(5, 1fr);
      align-items: center;
      height: var(--tabbar-h);
      padding: 0 var(--s2) env(safe-area-inset-bottom, 0px);
      background: color-mix(in srgb, var(--bg) 96%, transparent);
      border-top: 1px solid var(--line);
    }
    .tabbar a {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      gap: 3px;
      height: 56px;
      color: var(--ink-3);
      font-size: 11px;
      font-weight: 550;
      text-decoration: none;
    }
    .tabbar a[aria-current='page'] {
      color: var(--ink);
    }
    .tabbar a[aria-current='page'] span {
      text-decoration: underline 2px var(--mark);
      text-underline-offset: 4px;
    }
    .shutter {
      justify-self: center;
      display: grid;
      place-items: center;
      width: 58px;
      height: 58px;
      margin-top: -18px;
      border-radius: 50%;
      background: var(--bg);
    }
    .ring {
      display: grid;
      place-items: center;
      width: 52px;
      height: 52px;
      border: 2.5px solid var(--ink);
      border-radius: 50%;
    }
    .disc {
      width: 38px;
      height: 38px;
      border-radius: 50%;
      background: var(--ink);
      transition: transform 120ms var(--ease);
    }
    .shutter:active .disc {
      transform: scale(0.86);
    }
    .shutter.busy .disc {
      background: var(--mark);
      animation: shutter-busy 1.4s ease-in-out infinite;
    }
    @keyframes shutter-busy {
      50% {
        transform: scale(0.9);
      }
    }
  }
</style>
