<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    label: string;
    hint?: string;
    /** Id of the control the label belongs to. */
    htmlFor?: string;
    children: Snippet;
  }
  let { label, hint, htmlFor, children }: Props = $props();
</script>

<div class="adm-row">
  <div>
    {#if htmlFor}
      <label class="label" for={htmlFor}>{label}</label>
    {:else}
      <span class="label">{label}</span>
    {/if}
    {#if hint}<div class="hint">{hint}</div>{/if}
  </div>
  <div class="adm-ctl">{@render children()}</div>
</div>
