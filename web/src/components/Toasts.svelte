<script lang="ts">
  import { bottomBar, dismiss, hold, release, toasts } from '../lib/toast.svelte';
  import Button from './Button.svelte';

  /** Pause auto-dismiss while the pointer or keyboard focus is on a toast. */
  function pausable(node: HTMLElement, id: number) {
    const on = () => hold(id);
    const off = () => release(id);
    const events = ['pointerenter', 'focusin'] as const;
    const leave = ['pointerleave', 'focusout'] as const;
    for (const e of events) node.addEventListener(e, on);
    for (const e of leave) node.addEventListener(e, off);
    return {
      destroy() {
        for (const e of events) node.removeEventListener(e, on);
        for (const e of leave) node.removeEventListener(e, off);
      },
    };
  }
</script>

<div class="toasts" class:lifted={bottomBar.active} role="status" aria-live="polite">
  {#each toasts as t (t.id)}
    <div class="toast" use:pausable={t.id}>
      <span class="msg">{t.message}</span>
      {#if t.action}
        {@const action = t.action}
        <Button
          variant="inverse"
          size="sm"
          onclick={() => {
            dismiss(t.id);
            action.run();
          }}>{action.label}</Button
        >
      {/if}
    </div>
  {/each}
</div>

<style>
  .toasts {
    position: fixed;
    left: 50%;
    bottom: calc(var(--s4) + var(--tabbar-h));
    z-index: 60;
    transform: translateX(-50%);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--s2);
    width: max-content;
    max-width: calc(100vw - 2 * var(--s4));
    pointer-events: none;
    transition: bottom 200ms var(--ease);
  }
  .lifted {
    bottom: calc(var(--s4) + var(--tabbar-h) + 72px);
  }
  .toast {
    pointer-events: auto;
    display: flex;
    align-items: center;
    gap: var(--s3);
    min-height: 40px;
    padding: 6px 6px 6px 16px;
    border-radius: var(--r-ctl);
    background: var(--film);
    color: var(--print);
    font-size: var(--t-13);
    box-shadow:
      0 0 0 1px var(--film-line),
      var(--shadow);
    animation: toast-in 220ms var(--ease-out);
  }
  .toast:not(:has(:global(.btn))) {
    padding-right: 16px;
  }
  .toast :global(.btn) {
    color: var(--edge);
  }
  .msg {
    min-width: 0;
  }
  @keyframes toast-in {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
  }
</style>
