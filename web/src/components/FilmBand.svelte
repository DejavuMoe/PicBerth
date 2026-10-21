<script lang="ts">
  // A decorative strip of empty 3:2 frames with edge print, bleeding off both sides of its container.
  // Sprocket holes are the shared `.film` primitive, punched in the page colour.
  interface Props {
    frames?: number;
  }
  let { frames = 7 }: Props = $props();

  const numbers = $derived(Array.from({ length: frames }, (_, i) => i + 1));
</script>

<div class="film band" aria-hidden="true">
  <div class="track">
    {#each numbers as n (n)}
      <div class="cell">
        <span class="edge top-l">▸ {n}</span>
        <span class="edge top-r">{n}A</span>
        <div class="frame"></div>
        <span class="edge bot">PICBERTH 400</span>
      </div>
    {/each}
  </div>
</div>

<style>
  .band {
    padding-inline: 0;
    overflow: hidden;
    border-top: 1px solid var(--film-line);
    user-select: none;
  }
  .track {
    display: flex;
    gap: 6px;
    padding-left: 22px;
    width: max-content;
  }
  .cell {
    position: relative;
    flex: none;
    width: 120px;
    width: clamp(104px, 21cqi, 148px);
  }
  .frame {
    aspect-ratio: 3 / 2;
    border-radius: var(--r-thumb);
    background: var(--film-2);
  }
  .edge {
    position: absolute;
    color: var(--edge);
    pointer-events: none;
  }
  .top-l,
  .top-r {
    top: calc(-1 * var(--band) + 14px);
  }
  .top-l {
    left: 1px;
  }
  .top-r {
    right: 1px;
  }
  .bot {
    bottom: calc(-1 * var(--band) + 14px);
    left: 1px;
    color: var(--edge-2);
  }

  @media (max-width: 760px) {
    .cell {
      width: 84px;
    }
    .track {
      padding-left: 16px;
    }
  }
</style>
