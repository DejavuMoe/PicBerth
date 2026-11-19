<script lang="ts">
  import type { Library } from '../../lib/library.svelte';
  import Modal from '../Modal.svelte';

  let { lib }: { lib: Library } = $props();

  const fromLibrary = $derived(lib.mode === 'library');
  let confirm = $state<HTMLButtonElement>();
  const ids = $derived(lib.purgeIds ?? []);
  const single = $derived(ids.length === 1 ? lib.items.find((i) => i.id === ids[0]) : undefined);
  const title = $derived(
    ids.length > 1 ? `永久删除 ${ids.length} 张图片？` : single ? `永久删除“${single.filename}”？` : '永久删除图片？',
  );
  $effect(() => {
    if (lib.purgeFailed) confirm?.focus();
  });
</script>

<Modal {title} onclose={() => (lib.purgeIds = null)} dismissible={!lib.purgeBusy} descriptionId="purge-note">
  <p id="purge-note">{fromLibrary ? '图片删除后无法恢复。' : '文件和链接会立即失效，无法恢复。'}</p>
  {#if lib.purgeFailed}
    <p class="field-error" role="alert">永久删除失败，请重试。</p>
  {/if}
  {#snippet footer(close)}
    <button type="button" class="btn btn-secondary" data-autofocus disabled={lib.purgeBusy} onclick={close}>取消</button>
    <button type="button" class="btn btn-danger" bind:this={confirm} disabled={lib.purgeBusy} onclick={() => lib.confirmPurge()}>
      {lib.purgeBusy ? '正在删除…' : '永久删除'}
    </button>
  {/snippet}
</Modal>
