<script lang="ts">
  import { ICONS, type IconName } from '../lib/icons';

  interface Props {
    name: IconName;
    /** 's' renders the 14px variant (icon-s); default is 16px. */
    size?: 's';
  }
  let { name, size }: Props = $props();
</script>

<svg class="icon" class:icon-s={size === 's'} viewBox="0 0 16 16" aria-hidden="true" focusable="false">
  {#each ICONS[name] as shape, i (i)}
    {#if shape.t === 'path'}
      <path d={shape.d} />
    {:else if shape.t === 'circle'}
      <circle cx={shape.cx} cy={shape.cy} r={shape.r} />
    {:else}
      <rect x={shape.x} y={shape.y} width={shape.w} height={shape.h} rx={shape.rx} />
    {/if}
  {/each}
</svg>
