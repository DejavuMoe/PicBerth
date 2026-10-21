<script lang="ts">
  import type { Snippet } from 'svelte';
  import FilmBand from './FilmBand.svelte';

  interface Props {
    title: string;
    subtitle: string;
    children: Snippet;
  }
  let { title, subtitle, children }: Props = $props();
</script>

<div class="auth">
  <aside class="visual">
    <div class="lockup">
      <p class="wordmark display">PicBerth</p>
      <p class="tagline">图片的泊位</p>
    </div>
    <FilmBand />
  </aside>

  <main class="side">
    <div class="col">
      <h1 class="title display">{title}</h1>
      <p class="subtitle">{subtitle}</p>
      <div class="body">{@render children()}</div>
    </div>
  </main>
</div>

<style>
  .auth {
    display: grid;
    grid-template-columns: minmax(0, 44fr) minmax(0, 56fr);
    min-height: 100vh;
    min-height: 100dvh;
    background: var(--bg);
  }

  /* left: the film panel */
  .visual {
    container-type: inline-size;
    display: flex;
    flex-direction: column;
    min-width: 0;
    background: var(--film);
    color: var(--edge-2);
  }
  .lockup {
    margin-block: auto;
    padding: var(--s8) 40px;
  }
  .wordmark {
    color: var(--print);
    font-size: 88px;
    font-size: clamp(40px, 12.5cqi, 88px);
    line-height: 1;
    letter-spacing: -0.025em;
    white-space: nowrap;
    overflow-wrap: normal;
  }
  .tagline {
    margin-top: var(--s4);
    color: var(--edge-2);
    font-size: var(--t-16);
    letter-spacing: 0.32em;
  }

  /* right: the form */
  .side {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 0;
    padding: var(--s7) var(--gutter);
  }
  .col {
    width: 100%;
    max-width: 360px;
  }
  .title {
    font-size: var(--t-28);
    color: var(--ink);
  }
  .subtitle {
    margin-top: var(--s2);
    color: var(--ink-3);
  }
  .body {
    margin-top: var(--s5);
  }

  @media (max-width: 760px) {
    .auth {
      grid-template-columns: minmax(0, 1fr);
      grid-template-rows: auto 1fr;
    }
    .visual {
      min-height: 180px;
    }
    .lockup {
      display: flex;
      flex-wrap: wrap;
      align-items: baseline;
      justify-content: space-between;
      gap: var(--s1) var(--s3);
      margin-block: 0 auto;
      padding: var(--s4) var(--gutter) var(--s4);
    }
    .wordmark {
      font-size: clamp(32px, 9.5cqi, 44px);
    }
    .tagline {
      margin-top: 0;
      font-size: var(--t-13);
      letter-spacing: 0.24em;
    }
    .side {
      align-items: flex-start;
      padding-block: var(--s6) var(--s7);
    }
  }
</style>
