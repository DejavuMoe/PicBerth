<script lang="ts">
  import { copyText, INSECURE_HINT } from '../lib/clipboard';
  import { toast } from '../lib/toast.svelte';
  import Icon from './Icon.svelte';

  interface Props {
    text: string;
    label?: string;
    small?: boolean;
    variant?: 'secondary' | 'primary' | 'ghost';
    iconOnly?: boolean;
    /** Text control to select when the clipboard is unavailable. */
    field?: HTMLInputElement | null;
  }
  let { text, label = '复制', small = false, variant = 'secondary', iconOnly = false, field = null }: Props = $props();

  let done = $state(false);
  let timer: ReturnType<typeof setTimeout> | undefined;

  async function run(e: MouseEvent) {
    e.stopPropagation();
    const r = await copyText(text, field);
    if (r === 'failed') return toast('当前连接不安全，无法写入剪贴板');
    if (r === 'selected') return toast(INSECURE_HINT);
    done = true;
    clearTimeout(timer);
    timer = setTimeout(() => (done = false), 1400);
  }
  const name = $derived(done ? '已复制' : label);
</script>

<button
  type="button"
  class="btn btn-{variant}"
  class:btn-sm={small}
  class:btn-icon={iconOnly}
  onclick={run}
  aria-label={iconOnly ? name : undefined}
  title={iconOnly ? name : undefined}
>
  <Icon name={done ? 'check' : 'copy'} />
  {#if !iconOnly}<span>{name}</span>{/if}
</button>
