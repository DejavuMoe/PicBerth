<script lang="ts">
  interface Props {
    checked: boolean;
    onchange: (next: boolean) => void;
    label?: string;
    labelledby?: string;
    describedby?: string;
    disabled?: boolean;
  }
  let { checked, onchange, label, labelledby, describedby, disabled = false }: Props = $props();
</script>

<button
  type="button"
  role="switch"
  class="sw"
  class:on={checked}
  aria-checked={checked}
  aria-label={label}
  aria-labelledby={labelledby}
  aria-describedby={describedby}
  {disabled}
  onclick={() => onchange(!checked)}
>
  <span class="knob"></span>
</button>

<style>
  .sw {
    position: relative;
    flex: none;
    width: 36px;
    height: 20px;
    border-radius: 3px;
    border: 1px solid var(--line-strong);
    background: var(--surface-2);
    transition:
      background-color var(--dur) var(--ease),
      border-color var(--dur) var(--ease);
  }
  .knob {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 14px;
    height: 14px;
    border-radius: 1.5px;
    background: var(--ink-3);
    transition:
      transform 180ms var(--ease-out),
      background-color var(--dur) var(--ease);
  }
  .on {
    background: var(--ink);
    border-color: var(--ink);
  }
  .on .knob {
    transform: translateX(16px);
    background: var(--bg);
  }
  /* On film-dark surfaces (the lightbox). */
  :global(.on-film) .sw {
    border-color: var(--film-line);
    background: color-mix(in srgb, var(--print) 6%, transparent);
  }
  :global(.on-film) .knob {
    background: var(--edge-2);
  }
  :global(.on-film) .on {
    background: var(--print);
    border-color: var(--print);
  }
  :global(.on-film) .on .knob {
    background: var(--film);
  }
  .sw:disabled {
    opacity: 0.45;
  }
</style>
