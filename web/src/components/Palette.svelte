<script lang="ts">
  import { tick, untrack } from 'svelte';
  import { api, isAbort } from '../lib/api';
  import type { IconName } from '../lib/icons';
  import { thumbSrc } from '../lib/image';
  import { copyLink } from '../lib/library.svelte';
  import { wideNav } from '../lib/media.svelte';
  import { overlay } from '../lib/modal';
  import { openImage } from '../lib/open.svelte';
  import { MOD, closePalette, palette } from '../lib/palette.svelte';
  import { openPicker } from '../lib/picker';
  import { navigate } from '../lib/router.svelte';
  import { setTheme, theme, THEME_OPTIONS } from '../lib/theme.svelte';
  import { tree } from '../lib/tree.svelte';
  import type { Image } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Item {
    key: string;
    title: string;
    icon?: IconName | undefined;
    image?: Image;
    sub?: string;
    kbd?: string;
    run: () => void;
  }
  interface Group {
    label: string;
    items: Item[];
  }

  let q = $state('');
  let idx = $state(0);
  let images = $state<Image[]>([]);
  let listEl = $state<HTMLElement | null>(null);

  // Images are searched on the server, a moment after typing stops.
  let seq = 0;
  $effect(() => {
    const s = q.trim();
    if (!s) {
      images = [];
      return;
    }
    const my = ++seq;
    const ctrl = new AbortController();
    const t = setTimeout(async () => {
      try {
        const page = await api.listImages({ state: 'active', folder: 'root', recursive: true, q: s, limit: 6 }, ctrl.signal);
        if (my === seq) images = page.items;
      } catch (e) {
        if (!isAbort(e) && my === seq) images = [];
      }
    }, 150);
    return () => {
      clearTimeout(t);
      ctrl.abort();
    };
  });

  const NAV_ITEMS: { key: string; title: string; icon: IconName; to: string }[] = [
    { key: 'go-library', title: '图片', icon: 'image', to: '/library' },
    { key: 'go-trash', title: '回收站', icon: 'trash', to: '/trash' },
    { key: 'go-tokens', title: '令牌', icon: 'key', to: '/tokens' },
    { key: 'go-settings', title: '设置', icon: 'sliders', to: '/settings' },
    { key: 'go-status', title: '状态', icon: 'pulse', to: '/status' },
  ];
  const THEME_ICON: Record<string, IconName> = { system: 'monitor', light: 'sun', dark: 'moon' };

  async function newFolder() {
    // The tree lists the folders of the library; start the new one from there.
    if (await navigate('/library')) tree.createSignal++;
  }

  async function newToken() {
    if (await navigate('/tokens')) palette.newToken = true;
  }

  const commands = $derived.by((): Group[] => [
    {
      label: '操作',
      items: [
        { key: 'upload', icon: 'upload', title: '上传图片', kbd: 'U', run: () => openPicker() },
        ...(wideNav.matches
          ? [{ key: 'folder', icon: 'folder-plus' as IconName, title: '新建文件夹', run: () => void newFolder() }]
          : []),
        { key: 'token', icon: 'key', title: '新建令牌', run: () => void newToken() },
      ],
    },
    {
      label: '前往',
      items: [
        ...NAV_ITEMS.map((n) => ({ key: n.key, icon: n.icon, title: n.title, run: () => void navigate(n.to) })),
        { key: 'go-docs', icon: 'book' as IconName, title: 'API 文档', run: () => void window.open('/docs/api', '_blank', 'noopener') },
      ],
    },
    {
      label: '主题',
      items: THEME_OPTIONS.map((t) => ({
        key: 'theme-' + t.key,
        icon: THEME_ICON[t.key],
        title: t.label,
        sub: theme.choice === t.key ? '当前' : '',
        run: () => setTheme(t.key),
      })),
    },
  ]);

  const groups = $derived.by((): Group[] => {
    const s = q.trim().toLowerCase();
    const out: Group[] = [];
    if (s) {
      if (images.length) {
        out.push({
          label: '图片',
          items: images.map((im) => ({
            key: 'i' + im.id,
            image: im,
            title: im.filename,
            sub: tree.label(im.folderId ?? null),
            run: () => void openImage(im.id, im.folderId ?? null),
          })),
        });
      }
      const fs = tree.folders.filter((f) => f.name.toLowerCase().includes(s)).slice(0, 5);
      if (fs.length) {
        out.push({
          label: '文件夹',
          items: fs.map((f) => ({
            key: 'f' + f.id,
            icon: 'folder' as IconName,
            title: f.name,
            sub: tree.label(f.parentId ?? null),
            run: () => tree.openFolder(f.id),
          })),
        });
      }
    }
    for (const g of commands) {
      const items = g.items.filter((c) => !s || c.title.toLowerCase().includes(s));
      if (items.length) out.push({ label: g.label, items });
    }
    return out;
  });
  const flat = $derived(groups.flatMap((g) => g.items));
  const cur = $derived(flat[Math.min(idx, flat.length - 1)]);

  $effect(() => {
    q;
    untrack(() => (idx = 0));
  });
  $effect(() => {
    idx;
    flat.length;
    void tick().then(() => listEl?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' }));
  });

  function run(it: Item, alt: boolean) {
    closePalette();
    if (alt && it.image) void copyLink(it.image);
    else it.run();
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      idx = Math.min(flat.length - 1, idx + 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      idx = Math.max(0, idx - 1);
    } else if (e.key === 'Enter' && cur) {
      e.preventDefault();
      run(cur, e.metaKey || e.ctrlKey);
    }
  }
</script>

<div class="cmdk-scrim" role="presentation" onpointerdown={(e) => e.target === e.currentTarget && closePalette()}>
  <div
    class="cmdk"
    role="dialog"
    aria-modal="true"
    aria-label="命令面板"
    tabindex="-1"
    use:overlay={{ onescape: closePalette, blocksDrop: true }}
    {onkeydown}
  >
    <div class="cmdk-input">
      <Icon name="search" />
      <!-- svelte-ignore a11y_autofocus -->
      <input
        autofocus
        data-autofocus
        autocomplete="off"
        spellcheck="false"
        bind:value={q}
        placeholder="搜索图片、文件夹或操作"
        aria-label="搜索图片、文件夹或操作"
        role="combobox"
        aria-expanded="true"
        aria-controls="cmdk-list"
        aria-activedescendant={cur ? 'cmdk-' + cur.key : undefined}
      />
    </div>
    <div class="cmdk-list" id="cmdk-list" role="listbox" aria-label="命令面板" bind:this={listEl}>
      {#if flat.length === 0}<div class="cmdk-empty">没有匹配的结果</div>{/if}
      {#each groups as g (g.label)}
        <div role="group" aria-label={g.label}>
          <div class="cmdk-group" aria-hidden="true">{g.label}</div>
          {#each g.items as it (it.key)}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <div
              id={'cmdk-' + it.key}
              class="cmdk-item"
              role="option"
              tabindex="-1"
              aria-selected={it === cur}
              onpointermove={() => it !== cur && (idx = flat.indexOf(it))}
              onclick={(e) => run(it, e.metaKey || e.ctrlKey)}
            >
              {#if it.image}
                <img class="thumb" src={thumbSrc(it.image)} alt="" />
              {:else if it.icon}
                <Icon name={it.icon} />
              {/if}
              <span class="t ellipsis">{it.title}</span>
              {#if it.sub}<span class="sub">{it.sub}</span>{/if}
              {#if it.kbd}<kbd>{it.kbd}</kbd>{/if}
              {#if it.image && it === cur}<span class="sub hint">{MOD} ↵ 复制链接</span>{/if}
            </div>
          {/each}
        </div>
      {/each}
    </div>
  </div>
</div>
