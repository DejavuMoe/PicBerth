<script lang="ts">
  import Button from './Button.svelte';
  import Dialog from './Dialog.svelte';

  interface Props {
    open: boolean;
    title: string;
    message: string;
    confirmLabel: string;
    cancelLabel?: string;
    danger?: boolean;
    onconfirm: () => void | Promise<void>;
    oncancel?: () => void;
  }
  let { open = $bindable(), title, message, confirmLabel, cancelLabel = '取消', danger = false, onconfirm, oncancel }: Props =
    $props();

  let busy = $state(false);

  async function confirm() {
    busy = true;
    try {
      await onconfirm();
    } finally {
      busy = false;
      open = false;
    }
  }
</script>

<Dialog bind:open {title} width="400px" onclose={oncancel}>
  <p class="msg">{message}</p>
  {#snippet footer()}
    <Button autofocus onclick={() => { open = false; oncancel?.(); }}>{cancelLabel}</Button>
    <Button variant={danger ? 'danger' : 'primary'} disabled={busy} onclick={confirm}>{confirmLabel}</Button>
  {/snippet}
</Dialog>

<style>
  .msg {
    color: var(--ink-2);
  }
</style>
