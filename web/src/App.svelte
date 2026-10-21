<script lang="ts">
  import { onMount, tick } from 'svelte';
  import Button from './components/Button.svelte';
  import Logo from './components/Logo.svelte';
  import Safelight from './components/Safelight.svelte';
  import Toasts from './components/Toasts.svelte';
  import TopBar from './components/TopBar.svelte';
  import { errorMessage } from './lib/api';
  import { watchHealth } from './lib/health.svelte';
  import { isTypingTarget, modalOpen } from './lib/modal';
  import { openPicker, registerPicker } from './lib/picker';
  import { isKnownPath, navigate, router, safeNext } from './lib/router.svelte';
  import { loadSession, session } from './lib/session.svelte';
  import { acceptAttr, addFiles, installGlobalInput } from './lib/upload.svelte';
  import Login from './screens/Login.svelte';
  import NotFound from './screens/NotFound.svelte';
  import Settings from './screens/Settings.svelte';
  import Setup from './screens/Setup.svelte';
  import Sheet from './screens/Sheet.svelte';
  import Status from './screens/Status.svelte';
  import Tokens from './screens/Tokens.svelte';

  onMount(() => {
    void loadSession();
  });

  // What is on screen follows from session + path, so a wrong screen never flashes before a redirect.
  const view = $derived.by(() => {
    if (!session.loaded) return 'boot';
    if (session.failure) return 'error';
    const p = router.path;
    if (session.setupRequired) return p === '/setup' ? 'setup' : 'boot';
    if (!session.authenticated) return p === '/login' ? 'login' : 'boot';
    if (p === '/login' || p === '/setup') return 'boot';
    return 'app';
  });

  // Auth gating: setup first, then login (remembering where the user was), then back to `next`.
  $effect(() => {
    if (!session.loaded || session.failure) return;
    const p = router.path;
    if (session.setupRequired) {
      if (p !== '/setup') void navigate('/setup', { replace: true, force: true });
    } else if (!session.authenticated) {
      if (p !== '/login') {
        const here = p + router.search;
        const next = isKnownPath(p) && here !== '/' ? `?next=${encodeURIComponent(here)}` : '';
        void navigate(`/login${next}`, { replace: true, force: true });
      }
    } else if (p === '/login' || p === '/setup') {
      void navigate(safeNext(router.query.get('next')), { replace: true });
    }
  });

  const TITLES: Record<string, string> = {
    '/': '底片',
    '/library': '底片',
    '/trash': '回收站',
    '/tokens': '令牌',
    '/settings': '设置',
    '/status': '状态',
    '/login': '登录',
    '/setup': '初始化',
  };
  $effect(() => {
    const t = TITLES[router.path];
    document.title = t ? `${t} · PicBerth` : 'PicBerth';
  });

  const onSheet = $derived(router.path === '/' || router.path === '/library');

  // Window-wide drop and paste feed the global upload queue; the developing roll lives on the sheet.
  $effect(() => {
    if (view !== 'app') return;
    return installGlobalInput(() => {
      if (!onSheet) void navigate('/');
    });
  });

  $effect(() => {
    if (view !== 'app') return;
    return watchHealth();
  });

  // "Lights on": the short transition from the login screen into the app.
  let lightsOn = $state(false);
  let lastView = '';
  $effect(() => {
    const v = view;
    if ((lastView === 'login' || lastView === 'setup') && v === 'app') {
      lightsOn = true;
      setTimeout(() => (lightsOn = false), 700);
    }
    lastView = v;
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

  let picker = $state<HTMLInputElement | null>(null);
  $effect(() => {
    registerPicker(picker);
    return () => registerPicker(null);
  });
  function onpicked() {
    if (!picker) return;
    const files = Array.from(picker.files ?? []);
    picker.value = '';
    if (files.length === 0) return;
    addFiles(files);
    if (!onSheet) void navigate('/');
  }

  function onkeydown(e: KeyboardEvent) {
    if (view !== 'app' || e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey) return;
    if (isTypingTarget(e.target) || modalOpen()) return;
    if (e.key === 'u' || e.key === 'U') {
      e.preventDefault();
      openPicker();
    }
  }

  function retryBoot() {
    session.loaded = false;
    void loadSession();
  }
</script>

<svelte:window {onkeydown} />

{#if view === 'app'}
  <a class="skip" href="#main" onclick={(e) => { e.preventDefault(); main?.focus(); }}>跳到内容</a>
  <TopBar />
  <main id="main" tabindex="-1" bind:this={main}>
    {#key router.path === '/library' ? '/' : router.path}
      {#if onSheet}
        <Sheet mode="library" />
      {:else if router.path === '/trash'}
        <Sheet mode="trash" />
      {:else if router.path === '/tokens'}
        <Tokens />
      {:else if router.path === '/settings'}
        <Settings />
      {:else if router.path === '/status'}
        <Status />
      {:else}
        <NotFound />
      {/if}
    {/key}
  </main>
  <input bind:this={picker} type="file" multiple hidden accept={acceptAttr()} onchange={onpicked} />
  <Safelight />
{:else if view === 'login'}
  <Login />
{:else if view === 'setup'}
  <Setup />
{:else if view === 'error'}
  <div class="boot">
    <Logo />
    <p class="problem display">无法连接到 PicBerth</p>
    <p class="muted">{session.failure ? errorMessage(session.failure) : ''}</p>
    <Button variant="primary" onclick={retryBoot}>重试</Button>
  </div>
{:else}
  <div class="boot" aria-busy="true"><span class="pulse"><Logo markOnly /></span></div>
{/if}

{#if lightsOn}<div class="lights-on" aria-hidden="true"></div>{/if}

<Toasts />

<style>
  main {
    outline: none;
  }
  .skip {
    position: fixed;
    top: var(--s2);
    left: var(--s2);
    z-index: 100;
    padding: var(--s2) var(--s3);
    border-radius: var(--r-ctl);
    background: var(--ink);
    color: var(--bg);
    transform: translateY(-200%);
    text-decoration: none;
  }
  .skip:focus {
    transform: none;
  }
  .boot {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    justify-content: center;
    gap: var(--s3);
    min-height: 100dvh;
    padding: var(--s7) var(--gutter);
    max-width: 480px;
    margin: 0 auto;
  }
  .boot[aria-busy='true'] {
    align-items: center;
  }
  .pulse {
    animation: boot-pulse 1.2s ease-in-out infinite;
  }
  @keyframes boot-pulse {
    50% {
      opacity: 0.35;
    }
  }
  .problem {
    font-size: var(--t-28);
  }
  .lights-on {
    position: fixed;
    inset: 0;
    z-index: 90;
    pointer-events: none;
    background:
      radial-gradient(ellipse 70% 55% at 50% 0%, rgb(214 40 22 / 0.55), transparent 70%),
      #120806;
    animation: lights-on 650ms var(--ease) forwards;
  }
  @keyframes lights-on {
    0% {
      opacity: 1;
    }
    18% {
      opacity: 0.92;
    }
    24% {
      opacity: 1;
    }
    100% {
      opacity: 0;
    }
  }
</style>
