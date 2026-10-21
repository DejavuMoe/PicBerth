<script lang="ts" generics="T extends string">
  interface Tab {
    value: T;
    label: string;
  }
  interface Props {
    tabs: Tab[];
    value: T;
    onchange: (value: T) => void;
    label: string;
    /** Panels must use id="{idPrefix}-panel" and aria-labelledby="{idPrefix}-tab-{value}". */
    idPrefix: string;
  }
  let { tabs, value, onchange, label, idPrefix }: Props = $props();

  let buttons = $state<(HTMLButtonElement | null)[]>([]);

  function onkeydown(e: KeyboardEvent, i: number) {
    let next = -1;
    if (e.key === 'ArrowRight') next = (i + 1) % tabs.length;
    else if (e.key === 'ArrowLeft') next = (i - 1 + tabs.length) % tabs.length;
    else if (e.key === 'Home') next = 0;
    else if (e.key === 'End') next = tabs.length - 1;
    if (next < 0) return;
    e.preventDefault();
    const t = tabs[next];
    if (t) onchange(t.value);
    buttons[next]?.focus();
  }
</script>

<div class="tabs" role="tablist" aria-label={label}>
  {#each tabs as t, i (t.value)}
    <button
      type="button"
      role="tab"
      id="{idPrefix}-tab-{t.value}"
      aria-selected={t.value === value}
      aria-controls="{idPrefix}-panel"
      tabindex={t.value === value ? 0 : -1}
      class:on={t.value === value}
      bind:this={buttons[i]}
      onclick={() => onchange(t.value)}
      onkeydown={(e) => onkeydown(e, i)}
    >
      {t.label}
    </button>
  {/each}
</div>

<style>
  .tabs {
    display: flex;
    gap: var(--s4);
    border-bottom: 1px solid var(--line);
  }
  button {
    position: relative;
    height: var(--control-h);
    padding: 0 2px;
    margin-bottom: -1px;
    border-bottom: 2px solid transparent;
    color: var(--ink-3);
    font-weight: 500;
    transition: color var(--dur) var(--ease);
  }
  button:hover {
    color: var(--ink);
  }
  .on {
    color: var(--ink);
    border-bottom-color: var(--ink);
  }
</style>
