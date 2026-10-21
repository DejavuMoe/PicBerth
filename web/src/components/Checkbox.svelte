<script lang="ts">
  import Icon from './Icon.svelte';

  interface Props {
    checked: boolean;
    onchange: (next: boolean, e: Event) => void;
    /** Visible label. */
    label?: string;
    description?: string;
    /** Accessible name when there is no visible label. */
    ariaLabel?: string;
    disabled?: boolean;
    /** Ctrl/Shift-aware callers need the click event. */
    onclick?: (e: MouseEvent) => void;
  }
  let { checked, onchange, label, description, ariaLabel, disabled = false, onclick }: Props = $props();
</script>

<label class="cb" class:has-text={!!label}>
  <input
    type="checkbox"
    {checked}
    {disabled}
    aria-label={label ? undefined : ariaLabel}
    {onclick}
    onchange={(e) => onchange(e.currentTarget.checked, e)}
  />
  <span class="box" aria-hidden="true"><Icon name="check" size={14} /></span>
  {#if label}
    <span class="text">
      <span class="name">{label}</span>
      {#if description}<span class="desc">{description}</span>{/if}
    </span>
  {/if}
</label>

<style>
  .cb {
    position: relative;
    display: inline-flex;
    align-items: flex-start;
    gap: var(--s2);
    cursor: pointer;
  }
  input {
    position: absolute;
    inset: 0 auto auto 0;
    width: 18px;
    height: 18px;
    margin: 0;
    opacity: 0;
    cursor: pointer;
  }
  .box {
    display: grid;
    place-items: center;
    flex: none;
    width: 18px;
    height: 18px;
    margin-top: 1px;
    border: 1.5px solid var(--line-strong);
    border-radius: 2px;
    background: var(--surface);
    color: transparent;
    transition:
      background-color var(--dur) var(--ease),
      border-color var(--dur) var(--ease);
  }
  input:checked + .box {
    background: var(--ink);
    border-color: var(--ink);
    color: var(--bg);
  }
  input:focus-visible + .box {
    outline: 2px solid var(--ink);
    outline-offset: 2px;
  }
  input:disabled + .box {
    opacity: 0.45;
  }
  .text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .desc {
    color: var(--ink-3);
    font-size: var(--t-13);
  }
</style>
