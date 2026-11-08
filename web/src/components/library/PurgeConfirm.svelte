<script lang="ts">
  import type { Library } from '../../lib/library.svelte';
  import Modal from '../Modal.svelte';

  let { lib }: { lib: Library } = $props();

  const ids = $derived(lib.purgeIds ?? []);
  const single = $derived(ids.length === 1 ? lib.items.find((i) => i.id === ids[0]) : undefined);
  const title = $derived(
    ids.length > 1 ? `永久删除 ${ids.length} 张图片？` : single ? `永久删除“${single.filename}”？` : '永久删除图片？',
  );
</script>

<Modal {title} onclose={() => (lib.purgeIds = null)}>
  <p>文件和链接会立即失效，无法恢复。</p>
  {#snippet footer(close)}
    <button type="button" class="btn btn-secondary" data-autofocus onclick={close}>取消</button>
    <button type="button" class="btn btn-danger" onclick={() => lib.confirmPurge()}>永久删除</button>
  {/snippet}
</Modal>
