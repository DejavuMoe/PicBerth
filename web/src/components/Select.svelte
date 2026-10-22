<script lang="ts" generics="T extends string | number">
  import { tick } from 'svelte';
  import Floating from './Floating.svelte';
  import Icon from './Icon.svelte';

  interface Option {
    value: T;
    label: string;
  }
  interface Props {
    options: Option[];
    value: T;
    onchange: (value: T) => void;
    id?: string;
    /** Accessible name when no <label for> points at the button. */
    label?: string;
    disabled?: boolean;
    invalid?: boolean;
    class?: string;
  }
  let { options, value, onchange, id, label, disabled = false, invalid = false, class: className = '' }: Props = $props();

  const uid = $props.id();
  const listId = `${uid}-list`;
  const optId = (i: number) => `${uid}-opt-${i}`;

  let open = $state(false);
  let button = $state<HTMLButtonElement | null>(null);
  let layer = $state<HTMLElement | null>(null);
  let hi = $state(0);

  const current = $derived(options.find((o) => o.value === value));

  function show() {
    if (disabled) return;
    hi = Math.max(0, options.findIndex((o) => o.value === value));
    open = true;
  }
  function close(refocus = true) {
    open = false;
    if (refocus) button?.focus();
  }
  function pick(i: number) {
    const o = options[i];
    if (!o) return;
    close();
    if (o.value !== value) onchange(o.value);
  }

  // Focus the list when it opens and keep the highlighted option in view.
  $effect(() => {
    if (!layer) return;
    layer.focus({ preventScroll: true });
  });
  $effect(() => {
    const i = hi;
    if (!layer) return;
    void tick().then(() => layer?.querySelector(`#${CSS.escape(optId(i))}`)?.scrollIntoView({ block: 'nearest' }));
  });

  function onbuttonkeydown(e: KeyboardEvent) {
    if (open || disabled) return;
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      show();
    }
  }
  function onlistkeydown(e: KeyboardEvent) {
    const n = options.length;
    if (e.key === 'ArrowDown') hi = (hi + 1) % n;
    else if (e.key === 'ArrowUp') hi = (hi - 1 + n) % n;
    else if (e.key === 'Home') hi = 0;
    else if (e.key === 'End') hi = n - 1;
    else if (e.key === 'Enter' || e.key === ' ') pick(hi);
    else if (e.key === 'Tab') {
      close();
      return;
    } else return;
    e.preventDefault();
  }
</script>

<div class="menu-wrap block">
  <button
    bind:this={button}
    type="button"
    {id}
    class="select {className}"
    class:invalid
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-controls={open ? listId : undefined}
    aria-label={label}
    {disabled}
    onclick={() => (open ? close() : show())}
    onkeydown={onbuttonkeydown}
  >
    <span class="truncate">{current?.label ?? ''}</span>
    <Icon name="chevron-down" />
  </button>
</div>
{#if open}
  <Floating
    anchor={button}
    bind:el={layer}
    onclose={() => close()}
    matchWidth
    class="menu select-list"
    id={listId}
    role="listbox"
    tabindex={-1}
    aria-label={label}
    aria-activedescendant={optId(hi)}
    onkeydown={onlistkeydown}
  >
    {#each options as o, i (o.value)}
      <button
        type="button"
        role="option"
        tabindex="-1"
        id={optId(i)}
        class="menu-item"
        class:hi={i === hi}
        aria-selected={o.value === value}
        onmousedown={(e) => e.preventDefault()}
        onmouseenter={() => (hi = i)}
        onclick={() => pick(i)}
      >
        <span class="menu-tick">{#if o.value === value}<Icon name="check" />{/if}</span>
        <span class="truncate">{o.label}</span>
      </button>
    {/each}
  </Floating>
{/if}
