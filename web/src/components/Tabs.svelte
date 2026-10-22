<script lang="ts" generics="T extends string">
  interface Option {
    key: T;
    label: string;
  }
  interface Props {
    options: Option[];
    value: T;
    onchange: (key: T) => void;
    label: string;
  }
  let { options, value, onchange, label }: Props = $props();

  // Arrow keys move between the tabs without reaching the page-level image shortcuts.
  function move(e: KeyboardEvent) {
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return;
    e.stopPropagation();
    const i = options.findIndex((o) => o.key === value);
    const next = options[(i + (e.key === 'ArrowRight' ? 1 : -1) + options.length) % options.length];
    if (!next) return;
    onchange(next.key);
    e.currentTarget instanceof HTMLElement && e.currentTarget.querySelector<HTMLElement>(`[data-key="${CSS.escape(next.key)}"]`)?.focus();
  }
</script>

<!-- svelte-ignore a11y_interactive_supports_focus -->
<div class="tabs" role="tablist" aria-label={label} onkeydown={move}>
  {#each options as o (o.key)}
    <button
      type="button"
      role="tab"
      data-key={o.key}
      aria-selected={value === o.key}
      tabindex={value === o.key ? 0 : -1}
      onclick={() => onchange(o.key)}
    >
      {o.label}
    </button>
  {/each}
</div>
