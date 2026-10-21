<script lang="ts">
  // While files are dragged over the window the darkroom switches to its safelight.
  import { currentVisibility, uploads } from '../lib/upload.svelte';

  const count = $derived(uploads.drag.count);
  const vis = $derived(currentVisibility() === 'public' ? '公开' : '私有');
</script>

{#if uploads.drag.active}
  <div class="safelight" aria-hidden="true">
    <div class="lamp"></div>
    <div class="msg">
      <p class="big display">松手，开始冲洗</p>
      <p class="sub edge">
        {count > 0 ? `${count} 张` : '照片'} · 冲洗后设为{vis}
      </p>
    </div>
    <div class="tray" aria-hidden="true">
      {#each { length: Math.min(Math.max(count, 1), 5) } as _, i (i)}
        <span class="ghost" style:--i={i}></span>
      {/each}
    </div>
  </div>
{/if}

<style>
  .safelight {
    position: fixed;
    inset: 0;
    z-index: 80;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--s6);
    pointer-events: none;
    background:
      radial-gradient(ellipse 80% 60% at 50% -8%, rgb(214 40 22 / 0.5), transparent 68%),
      rgb(16 7 5 / 0.95);
    color: #f6d9cf;
    animation: lamp-on 260ms steps(1, end) both;
  }
  @keyframes lamp-on {
    0% {
      opacity: 0.6;
    }
    30% {
      opacity: 0.25;
    }
    55% {
      opacity: 1;
    }
  }
  /* The lamp itself: a narrow red slot at the top edge, throwing its light down the room. */
  .lamp {
    position: absolute;
    top: 0;
    left: 50%;
    width: min(360px, 60vw);
    height: 5px;
    transform: translateX(-50%);
    border-radius: 0 0 3px 3px;
    background: #ff5a3c;
    box-shadow:
      0 0 18px 4px rgb(255 70 40 / 0.85),
      0 0 140px 50px rgb(214 40 22 / 0.4);
  }
  .msg {
    text-align: center;
  }
  .big {
    font-size: clamp(32px, 6vw, 64px);
    color: #ffe9e1;
  }
  .sub {
    margin-top: var(--s3);
    font-size: 13px;
    color: rgb(255 190 170 / 0.8);
  }
  .tray {
    display: flex;
    gap: 10px;
    padding: 26px 10px;
    background: rgb(0 0 0 / 0.5);
    border-radius: 2px;
  }
  .ghost {
    width: 84px;
    aspect-ratio: 3 / 2;
    border: 1.5px dashed rgb(255 150 120 / 0.45);
    animation: ghost 1.6s var(--ease) infinite;
    animation-delay: calc(var(--i) * 120ms);
  }
  @keyframes ghost {
    50% {
      border-color: rgb(255 150 120 / 0.9);
      background: rgb(255 120 90 / 0.08);
    }
  }
  @media (max-width: 560px) {
    .ghost {
      width: 52px;
    }
  }
</style>
