<script lang="ts">
  interface Props {
    checked: boolean;
    label: string;
    disabled?: boolean;
    onchange: (value: boolean) => void;
  }
  let { checked, label, disabled = false, onchange }: Props = $props();
</script>

<button
  type="button"
  role="switch"
  class="switch"
  aria-checked={checked}
  aria-label={label}
  {disabled}
  onclick={() => onchange(!checked)}
></button>
