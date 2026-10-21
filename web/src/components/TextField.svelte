<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import type { IconName } from '../lib/icons';
  import Icon from './Icon.svelte';

  interface Props extends Omit<HTMLInputAttributes, 'class' | 'value' | 'children'> {
    label: string;
    value?: string;
    hint?: string;
    error?: string | null;
    /** Unit shown inside the field, after the text. */
    suffix?: string;
    icon?: IconName;
    mono?: boolean;
    /** One-time code: mono, letter-spaced, shown upper-case. */
    code?: boolean;
    hideLabel?: boolean;
    input?: HTMLInputElement | null;
    /** box (default) or line: a borderless field with only a bottom rule. */
    variant?: 'box' | 'line';
    /** Content at the end of the control (e.g. a keyboard hint). */
    trail?: import('svelte').Snippet;
  }

  let {
    label,
    value = $bindable(''),
    hint,
    error = null,
    suffix,
    icon,
    mono = false,
    code = false,
    hideLabel = false,
    input = $bindable(null),
    variant = 'box',
    trail,
    id: idProp,
    ...rest
  }: Props = $props();

  const uid = $props.id();
  const id = $derived(idProp ?? `f-${uid}`);
  const msgId = $derived(`${id}-msg`);
</script>

<div class="field" class:line={variant === 'line'} class:with-trail={!!trail}>
  <label for={id} class:sr-only={hideLabel}>{label}</label>
  <div class="control" class:invalid={!!error} class:with-icon={!!icon}>
    {#if icon}<span class="lead"><Icon name={icon} /></span>{/if}
    <input
      {id}
      bind:this={input}
      bind:value
      class:mono={mono || code}
      class:code
      aria-invalid={error ? 'true' : undefined}
      aria-describedby={error || hint ? msgId : undefined}
      {...rest}
    />
    {#if suffix}<span class="suffix mono">{suffix}</span>{/if}
    {#if trail}<span class="trail">{@render trail()}</span>{/if}
  </div>
  {#if error}
    <p id={msgId} class="msg error">{error}</p>
  {:else if hint}
    <p id={msgId} class="msg">{hint}</p>
  {/if}
</div>

<style>
  .field {
    display: flex;
    flex-direction: column;
    gap: var(--s1);
    min-width: 0;
  }
  label:not(.sr-only) {
    font-size: var(--t-13);
    font-weight: 500;
    color: var(--ink-2);
  }
  .control {
    display: flex;
    align-items: center;
    height: calc(var(--control-h) + 2px);
    border: 1px solid var(--line-strong);
    border-radius: var(--r-ctl);
    background: var(--surface);
    transition:
      border-color var(--dur) var(--ease),
      box-shadow var(--dur) var(--ease);
  }
  .control:hover {
    border-color: var(--ink-3);
  }
  .control:focus-within {
    border-color: var(--ink);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--ink) 12%, transparent);
  }
  /* "Sleeve label": no box, a bottom rule only. */
  .line .control {
    height: 48px;
    border-width: 0 0 1.5px;
    border-color: var(--line-strong);
    border-radius: 0;
    background: none;
  }
  .line .control:focus-within {
    border-color: var(--ink);
    box-shadow: none;
  }
  .line input {
    padding-inline: 2px;
    font-size: 17px;
  }
  .line .lead {
    padding-left: 0;
    margin-right: 6px;
  }
  .line.with-trail .control {
    padding-right: 0;
  }
  .trail {
    display: inline-flex;
    align-items: center;
    padding-inline: var(--s2);
  }
  .control.invalid,
  .control.invalid:hover {
    border-color: var(--danger);
  }
  .control.invalid:focus-within {
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--danger) 20%, transparent);
  }
  .lead {
    display: grid;
    place-items: center;
    padding-left: var(--s2);
    color: var(--ink-3);
  }
  input {
    flex: 1;
    min-width: 0;
    height: 100%;
    padding: 0 var(--s3);
    border: 0;
    background: none;
    outline: none;
    font-size: var(--t-14);
  }
  input:focus-visible {
    box-shadow: none;
  }
  .with-icon input {
    padding-left: var(--s2);
  }
  input.mono {
    font-family: var(--font-mono);
    font-size: var(--t-13);
  }
  input.code {
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .suffix {
    padding-right: var(--s3);
    color: var(--ink-3);
    font-size: var(--t-13);
  }
  .msg {
    font-size: var(--t-13);
    color: var(--ink-3);
  }
  .error {
    color: var(--danger);
  }
</style>
