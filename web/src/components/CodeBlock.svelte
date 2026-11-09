<script lang="ts">
  import { copyText } from '../lib/clipboard';

  let { text }: { text: string } = $props();
  let copied = $state(false);
  let timer: ReturnType<typeof setTimeout> | undefined;

  async function copy() {
    const r = await copyText(text);
    if (r === 'failed') return;
    copied = true;
    clearTimeout(timer);
    timer = setTimeout(() => (copied = false), 1500);
  }
</script>

<div class="code">
  <pre>{text}</pre>
  <button type="button" class="btn btn-ghost btn-sm copy" onclick={() => void copy()}>{copied ? '已复制' : '复制'}</button>
</div>
