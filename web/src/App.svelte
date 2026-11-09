<script lang="ts">
  import { onMount } from 'svelte';
  import DropOverlay from './components/DropOverlay.svelte';
  import Shell from './components/Shell.svelte';
  import Toasts from './components/Toasts.svelte';
  import { errorMessage } from './lib/api';
  import { refreshFacets } from './lib/facets.svelte';
  import { watchHealth } from './lib/health.svelte';
  import { isTypingTarget, modalOpen } from './lib/modal';
  import { openPicker, registerPicker } from './lib/picker';
  import { isKnownPath, isLibraryPath, navigate, router, safeNext } from './lib/router.svelte';
  import { loadSession, session } from './lib/session.svelte';
  import { acceptAttr, addFiles, installGlobalInput } from './lib/upload.svelte';
  import DocsApi from './screens/DocsApi.svelte';
  import Library from './screens/Library.svelte';
  import Login from './screens/Login.svelte';
  import NotFound from './screens/NotFound.svelte';
  import Settings from './screens/Settings.svelte';
  import Setup from './screens/Setup.svelte';
  import Status from './screens/Status.svelte';
  import Tokens from './screens/Tokens.svelte';

  onMount(() => {
    void loadSession();
  });

  // What is on screen follows from session + path, so a wrong screen never flashes before a redirect.
  // The API reference is public and needs no session.
  const view = $derived.by(() => {
    if (router.path === '/docs/api') return 'docs';
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
    if (p === '/docs/api') return; // public page, no session needed
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
    } else if (p === '/') {
      void navigate('/library', { replace: true });
    }
  });

  const TITLES: Record<string, string> = {
    '/login': '登录',
    '/setup': '初始化',
    '/trash': '回收站',
    '/tokens': '令牌',
    '/settings': '设置',
    '/status': '状态',
    '/docs/api': 'API 文档',
  };
  $effect(() => {
    const p = router.path;
    const t = p === '/' || isLibraryPath(p) ? '图片' : TITLES[p];
    document.title = t ? `${t} - PicBerth` : 'PicBerth';
  });

  // Window-wide drop and paste feed the upload queue, which opens the upload dialog.
  $effect(() => {
    if (view !== 'app') return;
    return installGlobalInput();
  });

  $effect(() => {
    if (view !== 'app') return;
    void refreshFacets();
    return watchHealth();
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
    addFiles(files);
  }

  function onkeydown(e: KeyboardEvent) {
    if (view !== 'app' || e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey) return;
    if (isTypingTarget(e.target) || modalOpen()) return;
    if (router.path !== '/' && !isLibraryPath(router.path)) return;
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
  <Shell>
    {#if router.path === '/' || isLibraryPath(router.path)}
      <Library mode="library" />
    {:else if router.path === '/trash'}
      <Library mode="trash" />
    {:else if router.path === '/tokens'}
      <Tokens />
    {:else if router.path === '/settings'}
      <Settings />
    {:else if router.path === '/status'}
      <Status />
    {:else}
      <NotFound />
    {/if}
    <DropOverlay />
  </Shell>
  <input bind:this={picker} type="file" multiple hidden accept={acceptAttr()} onchange={onpicked} />
{:else if view === 'docs'}
  <DocsApi />
{:else if view === 'login'}
  <Login />
{:else if view === 'setup'}
  <Setup />
{:else if view === 'error'}
  <div class="boot">
    <div class="brand">
      <span class="brand-mark" aria-hidden="true"></span>
      PicBerth
    </div>
    <strong class="strong">无法连接到 PicBerth</strong>
    <p class="muted">{session.failure ? errorMessage(session.failure) : ''}</p>
    <button type="button" class="btn btn-secondary" onclick={retryBoot}>重试</button>
  </div>
{:else}
  <div class="boot" aria-busy="true">
    <div class="brand pulse">
      <span class="brand-mark" aria-hidden="true"></span>
      PicBerth
    </div>
  </div>
{/if}

<Toasts />
