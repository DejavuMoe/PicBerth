<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    /** One line under the title. */
    meta?: string;
    /** Machine data (counts, sizes) is mono; prose meta is not. */
    mono?: boolean;
    /** A hero figure set next to the title (e.g. a frame count). */
    figure?: string;
    figureUnit?: string;
    actions?: Snippet;
  }
  let { title, meta, mono = false, figure, figureUnit, actions }: Props = $props();
</script>

<header class="ph">
  <div class="lead">
    <div class="titles">
      <h1 class="display">{title}</h1>
      {#if figure}
        <p class="figure"><span class="display n">{figure}</span>{#if figureUnit}<span class="u">{figureUnit}</span>{/if}</p>
      {/if}
    </div>
    {#if meta}<p class="meta" class:mono>{meta}</p>{/if}
  </div>
  {#if actions}
    <div class="actions">{@render actions()}</div>
  {/if}
</header>

<style>
  .ph {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    justify-content: space-between;
    gap: var(--s3) var(--s4);
    margin-bottom: var(--s5);
  }
  .lead {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
  }
  .titles {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: var(--s1) var(--s4);
  }
  h1 {
    font-size: var(--t-28);
  }
  .figure {
    display: inline-flex;
    align-items: baseline;
    gap: 6px;
    color: var(--ink-3);
  }
  .n {
    font-size: var(--t-28);
    color: var(--ink-3);
    font-weight: 600;
  }
  .u {
    font-size: var(--t-13);
  }
  .meta {
    color: var(--ink-3);
    font-size: var(--t-13);
    white-space: pre-wrap;
  }
  .actions {
    display: flex;
    align-items: center;
    gap: var(--s2);
  }
</style>
