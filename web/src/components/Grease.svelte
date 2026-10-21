<script lang="ts">
  // A red china-marker stroke, generated for the element's real size so it never looks stretched.
  // `box` circles a frame (selection), `line` underlines a word (current nav item).

  interface Props {
    shape: 'box' | 'line';
    /** Same seed, same hand: a frame keeps its mark when re-rendered. */
    seed?: string;
  }
  let { shape, seed = '' }: Props = $props();

  let w = $state(0);
  let h = $state(0);

  function rng(s: string) {
    let a = 2166136261;
    for (let i = 0; i < s.length; i++) a = Math.imul(a ^ s.charCodeAt(i), 16777619);
    return () => {
      a |= 0;
      a = (a + 0x6d2b79f5) | 0;
      let t = Math.imul(a ^ (a >>> 15), 1 | a);
      t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
  }

  const d = $derived.by(() => {
    if (w < 4 || h < 2) return '';
    const r = rng(seed + shape);
    const j = (a: number) => (r() - 0.5) * 2 * a;
    const f = (n: number) => n.toFixed(1);
    if (shape === 'line') {
      const y = h * 0.55;
      return `M${f(1 + j(1))} ${f(y + j(1.2))} Q${f(w * 0.45 + j(4))} ${f(y - 2.2 + j(1.2))} ${f(w - 1 + j(1))} ${f(y - 0.6 + j(1.2))}`;
    }
    const m = 4; // stroke stays inside the svg box
    const k = Math.min(10, w * 0.04);
    const tl = [m + j(3), m + j(3)];
    const tr = [w - m + j(3), m + j(3)];
    const br = [w - m + j(3), h - m + j(3)];
    const bl = [m + j(3), h - m + j(3)];
    // Start a little in on the top edge, go round clockwise, and overshoot the start along the top,
    // the way a hand closes a loop without lifting the pencil.
    const start = [tl[0]! + w * (0.08 + r() * 0.06), tl[1]! + 1 + j(2)];
    const end = [tl[0]! + w * (0.2 + r() * 0.12), tl[1]! + 2 + j(2.5)];
    const leftUp = [tl[0]! + j(2), tl[1]! + h * (0.22 + r() * 0.1)];
    const mid = (a: number[], b: number[], bow: number, vertical: boolean) =>
      vertical ? `${f((a[0]! + b[0]!) / 2 + bow)} ${f((a[1]! + b[1]!) / 2)}` : `${f((a[0]! + b[0]!) / 2)} ${f((a[1]! + b[1]!) / 2 + bow)}`;
    return [
      `M${f(start[0]!)} ${f(start[1]!)}`,
      `Q${mid(start, tr, j(k * 0.4) - k * 0.25, false)} ${f(tr[0]!)} ${f(tr[1]!)}`,
      `Q${mid(tr, br, j(k * 0.4) + k * 0.2, true)} ${f(br[0]!)} ${f(br[1]!)}`,
      `Q${mid(br, bl, j(k * 0.4) + k * 0.2, false)} ${f(bl[0]!)} ${f(bl[1]!)}`,
      `Q${mid(bl, leftUp, j(k * 0.3) - k * 0.15, true)} ${f(leftUp[0]!)} ${f(leftUp[1]!)}`,
      `Q${f(tl[0]! - 1 + j(1.5))} ${f(tl[1]! + j(1.5))} ${f(end[0]!)} ${f(end[1]!)}`,
    ].join(' ');
  });
</script>

<span class="grease-wrap {shape}" bind:clientWidth={w} bind:clientHeight={h} aria-hidden="true">
  {#if d}
    <svg width={w} height={h} viewBox="0 0 {w} {h}">
      <path class="grease" {d} pathLength="1" />
    </svg>
  {/if}
</span>

<style>
  .grease-wrap {
    position: absolute;
    display: block;
    pointer-events: none;
  }
  .box {
    inset: -7px;
    z-index: 3;
  }
  .line {
    left: -2px;
    right: -2px;
    bottom: -9px;
    height: 10px;
  }
  svg {
    display: block;
    overflow: visible;
  }
  .line :global(.grease) {
    stroke-width: 2.5;
  }
  .box :global(.grease) {
    stroke-width: 3.2;
    filter: drop-shadow(0 0 0.5px var(--mark));
  }
</style>
