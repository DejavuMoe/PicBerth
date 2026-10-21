<script lang="ts">
  import Icon from './Icon.svelte';

  interface Option {
    value: string;
    label: string;
  }
  interface Props {
    label: string;
    options: Option[];
    value: string;
    onchange?: (value: string) => void;
    hideLabel?: boolean;
    block?: boolean;
    quiet?: boolean;
  }
  let { label, options, value = $bindable(), onchange, hideLabel = false, block = false, quiet = false }: Props =
    $props();

  const uid = $props.id();
</script>

<div class="field" class:block class:quiet>
  <label for="sel-{uid}" class:sr-only={hideLabel}>{label}</label>
  <div class="control">
    <select id="sel-{uid}" bind:value onchange={() => onchange?.(value)}>
      {#each options as o (o.value)}
        <option value={o.value}>{o.label}</option>
      {/each}
    </select>
    <span class="chev"><Icon name="chevron-down" /></span>
  </div>
</div>

<style>
  .field {
    display: inline-flex;
    flex-direction: column;
    gap: var(--s1);
    min-width: 0;
  }
  .block {
    display: flex;
  }
  label:not(.sr-only) {
    font-size: var(--t-13);
    font-weight: 500;
    color: var(--ink-2);
  }
  .control {
    position: relative;
    display: flex;
  }
  select {
    appearance: none;
    width: 100%;
    height: var(--control-h);
    padding: 0 calc(var(--s3) + 20px) 0 var(--s3);
    border: 1px solid var(--line-strong);
    border-radius: var(--r-ctl);
    background: var(--surface);
    font-size: var(--t-13);
    font-weight: 550;
    cursor: pointer;
    transition: border-color var(--dur) var(--ease);
  }
  select:hover:not(:focus-visible) {
    border-color: var(--ink);
  }
  /* Quiet: text-only until hovered (toolbars). */
  .quiet select {
    border-color: transparent;
    background: none;
    padding-left: var(--s2);
    color: var(--ink-2);
  }
  .quiet select:hover:not(:focus-visible) {
    border-color: var(--line-strong);
    color: var(--ink);
  }
  .chev {
    position: absolute;
    top: 0;
    right: var(--s2);
    bottom: 0;
    display: grid;
    place-items: center;
    color: var(--ink-3);
    pointer-events: none;
  }
</style>
