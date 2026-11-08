<script lang="ts" generics="T extends string | number">
  import Icon from './Icon.svelte';
  import Menu from './Menu.svelte';
  import MenuItem from './MenuItem.svelte';

  interface Option {
    value: T;
    label: string;
  }
  interface Props {
    options: Option[];
    value: T;
    onchange: (value: T) => void;
    /** Accessible name when no <label for> points at the button. */
    label?: string;
    /** Fixed width of the closed button in px. */
    width?: number;
    disabled?: boolean;
  }
  let { options, value, onchange, label, width, disabled = false }: Props = $props();

  const current = $derived(options.find((o) => o.value === value));
</script>

<Menu align="left" {width} {label}>
  {#snippet trigger(open, toggle)}
    <button
      type="button"
      class="btn btn-secondary"
      style:min-width={width ? `${width}px` : undefined}
      style:justify-content="space-between"
      style:font-weight="var(--w-regular)"
      aria-label={label}
      aria-haspopup="menu"
      aria-expanded={open}
      {disabled}
      onclick={toggle}
    >
      <span class="ellipsis">{current?.label ?? ''}</span>
      <Icon name="down" size="s" />
    </button>
  {/snippet}
  {#snippet children(close)}
    {#each options as o (o.value)}
      <MenuItem checked={o.value === value} disabled={disabled} onclick={() => (onchange(o.value), close())}>
        {o.label}
      </MenuItem>
    {/each}
  {/snippet}
</Menu>
