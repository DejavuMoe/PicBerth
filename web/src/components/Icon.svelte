<script lang="ts">
  import { ICONS, type IconName } from '../lib/icons';

  interface Props {
    name: IconName;
    size?: number;
  }
  let { name, size = 16 }: Props = $props();
  const shape = $derived(ICONS[name] as { d: string[]; dots?: [number, number, number][] });
</script>

<svg
  class="icon"
  width={size}
  height={size}
  viewBox="0 0 16 16"
  fill="none"
  stroke="currentColor"
  stroke-width="1.5"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
  focusable="false"
>
  {#each shape.d as d, i (i)}
    <path {d} />
  {/each}
  {#each shape.dots ?? [] as [cx, cy, r], i (i)}
    <circle {cx} {cy} {r} fill="currentColor" stroke="none" />
  {/each}
</svg>

<style>
  .icon {
    flex: none;
    display: block;
  }
</style>
