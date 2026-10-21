<script lang="ts">
  import { bottomBar, dismiss, hold, release, toasts } from '../lib/toast.svelte';

  /** Pause auto-dismiss while the pointer or keyboard focus is on a toast. */
  function pausable(node: HTMLElement, id: number) {
    const on = () => hold(id);
    const off = () => release(id);
    node.addEventListener('pointerenter', on);
    node.addEventListener('focusin', on);
    node.addEventListener('pointerleave', off);
    node.addEventListener('focusout', off);
    return {
      destroy() {
        node.removeEventListener('pointerenter', on);
        node.removeEventListener('focusin', on);
        node.removeEventListener('pointerleave', off);
        node.removeEventListener('focusout', off);
      },
    };
  }
</script>

<div class="toasts" class:lifted={bottomBar.active} role="status" aria-live="polite">
  {#each toasts as t (t.id)}
    <div class="toast" class:has-action={!!t.action} use:pausable={t.id}>
      <span>{t.message}</span>
      {#if t.action}
        {@const action = t.action}
        <button
          type="button"
          onclick={() => {
            dismiss(t.id);
            action.run();
          }}>{action.label}</button
        >
      {/if}
    </div>
  {/each}
</div>
