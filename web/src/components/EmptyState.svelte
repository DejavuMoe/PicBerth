<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    message: string;
    /** Optional larger headline above the message. */
    title?: string;
    children?: Snippet;
  }
  let { message, title, children }: Props = $props();
</script>

<div class="empty">
  {#if title}<p class="title display">{title}</p>{/if}
  <p class="msg">{message}</p>
  {#if children}
    <div class="actions">{@render children()}</div>
  {/if}
</div>

<style>
  .empty {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--s3);
    padding: var(--s7) 0;
    color: var(--ink-2);
  }
  .title {
    font-size: var(--t-20);
    color: var(--ink);
  }
  .actions {
    display: flex;
    gap: var(--s2);
    margin-top: var(--s1);
  }
</style>
