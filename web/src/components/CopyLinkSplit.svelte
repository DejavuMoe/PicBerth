<script lang="ts">
  import { copyText } from '../lib/clipboard';
  import { toast } from '../lib/toast.svelte';
  import type { Image } from '../lib/types';
  import { LINK_FORMATS, setLinkFormat, uploads } from '../lib/upload.svelte';
  import CopyButton from './CopyButton.svelte';
  import Icon from './Icon.svelte';
  import Menu from './Menu.svelte';
  import MenuItem from './MenuItem.svelte';

  let { image }: { image: Pick<Image, 'links'> } = $props();

  const label = $derived(LINK_FORMATS.find((f) => f.value === uploads.format)?.label ?? '');

  async function pick(f: (typeof LINK_FORMATS)[number], close: () => void) {
    close();
    setLinkFormat(f.value);
    const r = await copyText(image.links[f.value]);
    if (r !== 'copied') toast('当前连接不安全，无法写入剪贴板');
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
<div class="btn-group" onclick={(e) => e.stopPropagation()}>
  <CopyButton small text={image.links[uploads.format]} label={`复制 ${label}`} />
  <Menu align="right">
    {#snippet trigger(open, toggle)}
      <button type="button" class="btn btn-secondary btn-icon btn-sm" aria-label="选择链接格式" aria-expanded={open} onclick={toggle}>
        <Icon name="chevron-down" size={14} />
      </button>
    {/snippet}
    {#snippet children(close)}
      <div class="menu-label">复制为</div>
      {#each LINK_FORMATS as f (f.value)}
        <MenuItem icon={f.value === uploads.format ? 'check' : undefined} onclick={() => pick(f, close)}>{f.label}</MenuItem>
      {/each}
    {/snippet}
  </Menu>
</div>
