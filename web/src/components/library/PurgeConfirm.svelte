<script lang="ts">
  import Modal from '../Modal.svelte';

  let { count, oncancel, onconfirm }: { count: number; oncancel: () => void; onconfirm: () => void } = $props();
</script>

<Modal title={count === 1 ? '永久删除这张图片？' : `永久删除 ${count} 张图片？`} onclose={oncancel} width="420px">
  <p>文件会立即从磁盘删除，已发出的链接将失效。此操作无法撤销。</p>
  {#snippet footer()}
    <button type="button" class="btn btn-secondary" data-autofocus onclick={oncancel}>取消</button>
    <button type="button" class="btn btn-danger-solid" onclick={onconfirm}>永久删除</button>
  {/snippet}
</Modal>
