<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    label: string;
    /** Id of the control the label belongs to. */
    htmlFor?: string;
    children: Snippet;
  }
  let { label, htmlFor, children }: Props = $props();
</script>

<!-- Hairline row: label on the left, the control (and its error) on the right. -->
<div class="adm-row">
  {#if htmlFor}
    <label class="label" for={htmlFor}>{label}</label>
  {:else}
    <span class="label">{label}</span>
  {/if}
  <div class="adm-ctl">{@render children()}</div>
</div>
