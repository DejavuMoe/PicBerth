<script lang="ts">
  interface Props {
    /** Hide the wordmark (compact headers). */
    markOnly?: boolean;
  }
  let { markOnly = false }: Props = $props();
</script>

<span class="logo">
  <!-- A single film frame: the picture, moored between two rows of sprocket holes. -->
  <svg class="mark" viewBox="0 0 32 32" width="24" height="24" aria-hidden="true" focusable="false">
    <rect class="base" x="1" y="5" width="30" height="22" rx="2" />
    <g class="holes">
      {#each [4, 10.3, 16.6, 22.9] as x (x)}
        <rect {x} y="7" width="3" height="2.4" rx=".6" />
        <rect {x} y="22.6" width="3" height="2.4" rx=".6" />
      {/each}
      <rect x="5" y="11.4" width="22" height="9.2" rx=".8" />
    </g>
    <path class="line" d="M7 18.2 12.4 14l3.8 3 3.2-2.2 5.6 3.4" />
  </svg>
  {#if !markOnly}<span class="word display">PicBerth</span>{/if}
</span>

<style>
  .logo {
    display: inline-flex;
    align-items: center;
    gap: 9px;
  }
  .mark {
    flex: none;
    display: block;
  }
  .base {
    fill: var(--ink);
  }
  .holes {
    fill: var(--bg);
  }
  .line {
    fill: none;
    stroke: var(--mark);
    stroke-width: 1.6;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .word {
    font-size: 17px;
    line-height: 1;
    letter-spacing: -0.02em;
  }
</style>
