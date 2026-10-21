<script lang="ts">
  import type { Snippet } from 'svelte';
  import { bottomBar } from '../lib/toast.svelte';
  import IconButton from './IconButton.svelte';

  interface Props {
    count: number;
    onclear: () => void;
    children: Snippet;
    /** Noun after the count. */
    unit?: string;
  }
  let { count, onclear, children, unit = '帧已圈选' }: Props = $props();

  $effect(() => {
    bottomBar.active = true;
    return () => {
      bottomBar.active = false;
    };
  });
</script>

<div class="bar on-film" role="region" aria-label="批量操作">
  <span class="count" aria-live="polite">
    <span class="n display">{count}</span>
    <span class="unit">{unit}</span>
  </span>
  <span class="rule" aria-hidden="true"></span>
  <div class="actions">{@render children()}</div>
  <IconButton variant="inverse" icon="close" label="取消选择 (Esc)" onclick={onclear} />
</div>

<style>
  .bar {
    position: fixed;
    left: 50%;
    bottom: calc(var(--s4) + var(--tabbar-h));
    z-index: 40;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: var(--s2);
    width: max-content;
    max-width: min(820px, calc(100vw - 2 * var(--s3)));
    min-height: 56px;
    padding: 6px 6px 6px 18px;
    border-radius: var(--r-ctl);
    background: var(--film);
    color: var(--print);
    font-size: var(--t-13);
    box-shadow:
      0 0 0 1px var(--film-line),
      var(--shadow);
    animation: bar-in 240ms var(--ease-out);
  }
  .count {
    display: inline-flex;
    align-items: baseline;
    gap: 6px;
    flex: none;
  }
  .n {
    font-size: 26px;
    color: var(--mark);
  }
  .unit {
    color: var(--edge-2);
    font-size: var(--t-12);
  }
  .rule {
    flex: none;
    width: 1px;
    height: 24px;
    margin: 0 var(--s2);
    background: var(--film-line);
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 2px;
    min-width: 0;
  }
  @keyframes bar-in {
    from {
      opacity: 0;
      transform: translate(-50%, 14px);
    }
  }
  @media (max-width: 560px) {
    .bar {
      padding-left: 12px;
    }
    .rule,
    .unit {
      display: none;
    }
  }
</style>
