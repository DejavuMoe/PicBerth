<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    label: string;
    help?: string;
    /** id for the label element when the control is not a labelled input (groups, switches). */
    labelId?: string;
    children: Snippet;
  }
  let { label, help, labelId, children }: Props = $props();
</script>

<div class="row">
  <div class="lab">
    <p class="name" id={labelId}>{label}</p>
    {#if help}<p class="help">{help}</p>{/if}
  </div>
  <div class="ctl">{@render children()}</div>
</div>

<style>
  .row {
    display: grid;
    grid-template-columns: 260px minmax(0, 1fr);
    gap: var(--s2) var(--s6);
    padding: 18px 0;
    border-bottom: 1px solid var(--line);
  }
  .name {
    font-size: var(--t-14);
    font-weight: 600;
    line-height: 1.4;
  }
  .help {
    margin-top: 3px;
    color: var(--ink-3);
    font-size: var(--t-13);
    line-height: 1.5;
  }
  .ctl {
    min-width: 0;
  }
  @media (max-width: 720px) {
    .row {
      grid-template-columns: minmax(0, 1fr);
      gap: var(--s3);
    }
  }
</style>
