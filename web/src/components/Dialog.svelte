<script lang="ts">
  import type { Snippet } from 'svelte';
  import { closeOnBackdrop, lockScroll } from '../lib/modal';
  import IconButton from './IconButton.svelte';

  interface Props {
    open?: boolean;
    title: string;
    /** false: neither Esc, backdrop nor the close button dismiss it (e.g. a one-time secret is shown). */
    dismissible?: boolean;
    width?: string;
    onclose?: () => void;
    children: Snippet;
    footer?: Snippet;
  }
  let { open = $bindable(false), title, dismissible = true, width = '440px', onclose, children, footer }: Props = $props();

  const uid = $props.id();
  let el = $state<HTMLDialogElement | null>(null);

  $effect(() => {
    if (!el || !open) return;
    const dialog = el;
    if (!dialog.open) dialog.showModal();
    const unlock = lockScroll();
    const stopBackdrop = closeOnBackdrop(dialog, () => {
      if (dismissible) close();
    });
    return () => {
      stopBackdrop();
      unlock();
      if (dialog.open) dialog.close();
    };
  });

  function close() {
    open = false;
    onclose?.();
  }

  function oncancel(e: Event) {
    // Esc: keep the dialog in sync with state instead of letting the browser close it behind our back.
    e.preventDefault();
    if (dismissible) close();
  }

  // Some browsers close a modal dialog on a repeated Esc even after the cancel event was prevented.
  function onnativeclose() {
    if (!open) return;
    if (dismissible) close();
    else el?.showModal();
  }
</script>

<dialog bind:this={el} class="dlg" style:--w={width} aria-labelledby="dlg-{uid}" {oncancel} onclose={onnativeclose}>
  {#if open}
    <div class="box">
      <header>
        <h2 id="dlg-{uid}">{title}</h2>
        {#if dismissible}
          <IconButton icon="close" label="关闭" size="sm" onclick={close} />
        {/if}
      </header>
      <div class="body">{@render children()}</div>
      {#if footer}
        <footer>{@render footer()}</footer>
      {/if}
    </div>
  {/if}
</dialog>

<style>
  .dlg {
    width: min(var(--w), calc(100vw - 2 * var(--s4)));
    margin: auto;
    padding: 0;
    border: 1px solid var(--line-strong);
    border-radius: var(--r-dialog);
    background: var(--surface);
    box-shadow: var(--shadow);
  }
  .dlg[open] {
    animation: dlg-in 200ms var(--ease-out);
  }
  .dlg::backdrop {
    animation: scrim-in 200ms var(--ease);
  }
  @keyframes scrim-in {
    from {
      opacity: 0;
    }
  }
  .box {
    display: flex;
    flex-direction: column;
    max-height: calc(100dvh - 2 * var(--s4));
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--s3);
    padding: var(--s4) var(--s3) var(--s3) var(--s5);
  }
  h2 {
    font-family: var(--font-display);
    font-stretch: 112%;
    font-size: var(--t-20);
    font-weight: 750;
    letter-spacing: -0.01em;
  }
  .body {
    padding: var(--s2) var(--s5) var(--s5);
    overflow-y: auto;
  }
  footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--s2);
    padding: var(--s3) var(--s5);
    border-top: 1px solid var(--line);
  }
  @keyframes dlg-in {
    from {
      opacity: 0;
      transform: translateY(10px) scale(0.985);
    }
  }
</style>
