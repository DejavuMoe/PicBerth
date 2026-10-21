<script lang="ts" generics="T extends string">
  interface Option {
    value: T;
    label: string;
  }
  interface Props {
    options: Option[];
    value: T;
    onchange: (value: T) => void;
    /** Accessible name of the group. */
    label: string;
  }
  let { options, value, onchange, label }: Props = $props();

  let buttons = $state<(HTMLButtonElement | null)[]>([]);

  function onkeydown(e: KeyboardEvent, i: number) {
    let next = -1;
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') next = (i + 1) % options.length;
    else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') next = (i - 1 + options.length) % options.length;
    else if (e.key === 'Home') next = 0;
    else if (e.key === 'End') next = options.length - 1;
    if (next < 0) return;
    e.preventDefault();
    const opt = options[next];
    if (opt) onchange(opt.value);
    buttons[next]?.focus();
  }
</script>

<div class="seg" role="radiogroup" aria-label={label}>
  {#each options as opt, i (opt.value)}
    <button
      type="button"
      role="radio"
      aria-checked={opt.value === value}
      tabindex={opt.value === value ? 0 : -1}
      class:on={opt.value === value}
      bind:this={buttons[i]}
      onclick={() => onchange(opt.value)}
      onkeydown={(e) => onkeydown(e, i)}
    >
      {opt.label}
    </button>
  {/each}
</div>

<style>
  .seg {
    display: inline-flex;
    border: 1px solid var(--line-strong);
    border-radius: var(--r-ctl);
    overflow: hidden;
  }
  button {
    height: calc(var(--control-h) - 4px);
    padding: 0 12px;
    font-size: var(--t-13);
    font-weight: 550;
    color: var(--ink-2);
    white-space: nowrap;
    transition:
      background-color var(--dur) var(--ease),
      color var(--dur) var(--ease);
  }
  button + button {
    border-left: 1px solid var(--line-strong);
  }
  button:hover:not(.on) {
    color: var(--ink);
    background: var(--surface);
  }
  .on {
    background: var(--ink);
    color: var(--bg);
  }
  button:focus-visible {
    outline-offset: -3px;
    outline-color: var(--mark);
  }
  /* On film-dark surfaces. */
  :global(.on-film) .seg {
    border-color: var(--film-line);
  }
  :global(.on-film) button {
    color: var(--edge-2);
  }
  :global(.on-film) button + button {
    border-left-color: var(--film-line);
  }
  :global(.on-film) button:hover:not(.on) {
    background: color-mix(in srgb, var(--print) 8%, transparent);
    color: var(--print);
  }
  :global(.on-film) .on {
    background: var(--print);
    color: var(--film);
  }
</style>
