<script module lang="ts">
  export interface ComboOption {
    value: string;
    label: string;
    /** Shown at the end of the row (e.g. how many images carry the tag). */
    count?: number;
  }
</script>

<script lang="ts">
  import { tick, type Snippet } from 'svelte';
  import Floating from './Floating.svelte';
  import Icon from './Icon.svelte';

  interface Props {
    /** Renders the button that opens the popover. */
    trigger: Snippet<[boolean, () => void]>;
    options: ComboOption[];
    /** Called with the option value, or the cleaned text for a created entry. */
    onpick: (value: string) => void;
    /** Returns the cleaned text that may be created from what was typed, or '' when nothing may be created. */
    normalize?: (text: string) => string;
    /** Whether a created value already exists elsewhere (it is then not offered again). */
    taken?: (value: string) => boolean;
    /** Backspace in the empty field (e.g. to remove the last tag). */
    onemptybackspace?: () => void;
    /** Accessible name of the filter field. */
    label: string;
    placeholder?: string;
    maxlength?: number;
    align?: 'left' | 'right';
    up?: boolean;
  }
  let { trigger, options, onpick, normalize, taken, onemptybackspace, label, placeholder, maxlength = 64, align = 'left', up = false }: Props = $props();

  const uid = $props.id();
  const listId = `${uid}-list`;
  const optId = (i: number) => `${uid}-opt-${i}`;

  let open = $state(false);
  let text = $state('');
  let hi = $state(0);
  let wrap = $state<HTMLElement | null>(null);
  let layer = $state<HTMLElement | null>(null);
  let input = $state<HTMLInputElement | null>(null);

  function close() {
    if (layer?.contains(document.activeElement)) {
      (wrap?.querySelector<HTMLElement>('[aria-expanded]') ?? wrap?.querySelector<HTMLElement>('button'))?.focus();
    }
    open = false;
    text = '';
  }
  const toggle = () => (open ? close() : (open = true));

  type Row = { kind: 'opt' | 'new'; value: string; label: string; count?: number | undefined };
  const rows = $derived.by((): Row[] => {
    const cand = normalize ? normalize(text) : '';
    const needle = (cand || text.trim()).toLowerCase();
    const matches: Row[] = options
      .filter((o) => !needle || o.label.toLowerCase().includes(needle))
      .map((o) => ({ kind: 'opt', value: o.value, label: o.label, count: o.count }));
    const exact = cand !== '' && ((taken?.(cand) ?? false) || options.some((o) => o.label.toLowerCase() === cand.toLowerCase()));
    return cand && !exact ? [...matches, { kind: 'new', value: cand, label: cand }] : matches;
  });
  const cur = $derived(Math.min(hi, rows.length - 1));

  $effect(() => {
    text;
    hi = 0;
  });
  $effect(() => {
    if (open && input) input.focus({ preventScroll: true });
  });
  $effect(() => {
    const i = cur;
    if (!layer || i < 0) return;
    void tick().then(() => layer?.querySelector(`#${CSS.escape(optId(i))}`)?.scrollIntoView({ block: 'nearest' }));
  });

  function pick(row: Row | undefined) {
    if (!row) return;
    onpick(row.value);
    text = '';
    hi = 0;
  }
  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      if (rows.length) hi = (cur + (e.key === 'ArrowDown' ? 1 : -1) + rows.length) % rows.length;
    } else if (e.key === 'Enter') {
      e.preventDefault();
      pick(rows[cur]);
    } else if (e.key === 'Backspace' && !text) {
      onemptybackspace?.();
    } else if (e.key === 'Tab') {
      close();
    }
  }
</script>

<div class="menu-wrap" bind:this={wrap}>
  {@render trigger(open, toggle)}
</div>
{#if open}
  <Floating anchor={wrap} bind:el={layer} onclose={close} {align} {up} class="pop">
    <input
      bind:this={input}
      bind:value={text}
      class="pop-input"
      role="combobox"
      aria-expanded="true"
      aria-controls={rows.length ? listId : undefined}
      aria-activedescendant={rows.length ? optId(cur) : undefined}
      aria-autocomplete="list"
      aria-label={label}
      {placeholder}
      {maxlength}
      autocomplete="off"
      spellcheck="false"
      {onkeydown}
    />
    {#if rows.length > 0}
      <div class="pop-list" id={listId} role="listbox" aria-label={label}>
        {#each rows as row, i (`${row.kind}-${row.value}`)}
          <button
            type="button"
            role="option"
            tabindex="-1"
            id={optId(i)}
            class="opt"
            class:hi={i === cur}
            aria-selected={i === cur}
            onmousedown={(e) => e.preventDefault()}
            onmouseenter={() => (hi = i)}
            onclick={() => pick(row)}
          >
            {#if row.kind === 'new'}<Icon name="plus" />{/if}
            <span class="truncate">{row.label}</span>
            {#if row.kind === 'opt' && row.count !== undefined}<span class="count">{row.count}</span>{/if}
          </button>
        {/each}
      </div>
    {/if}
  </Floating>
{/if}
