<script lang="ts">
  import { onDestroy } from 'svelte';
  import { copyText } from '../lib/clipboard';
  import Button from './Button.svelte';

  interface Props {
    /** What the ticket admits: the token's name. */
    name: string;
    /** Human-readable scope labels, printed along the bottom as edge text. */
    scopes: string[];
    /** The secret printed on the ticket. */
    secret: string;
  }
  let { name, scopes, secret }: Props = $props();

  let code = $state<HTMLElement | null>(null);
  let copied = $state(false);
  let failed = $state(false);
  let timer: ReturnType<typeof setTimeout> | undefined;

  function selectSecret() {
    if (!code) return;
    const sel = window.getSelection();
    if (!sel) return;
    const range = document.createRange();
    range.selectNodeContents(code);
    sel.removeAllRanges();
    sel.addRange(range);
  }

  async function copy() {
    const r = await copyText(secret);
    clearTimeout(timer);
    failed = r === 'failed';
    copied = r === 'copied';
    if (failed) selectSecret();
    timer = setTimeout(() => {
      copied = false;
      failed = false;
    }, 1800);
  }
  onDestroy(() => clearTimeout(timer));
</script>

<div class="ticket" role="group" aria-label="新令牌：{name}">
  <div class="stub" aria-hidden="true">
    <span class="pb display">PB</span>
    <span class="edge admit">ADMIT ONE</span>
  </div>
  <div class="main">
    <div class="head">
      <p class="name">{name}</p>
      {#if scopes.length}<p class="edge scopes">{scopes.join(' / ')}</p>{/if}
    </div>
    <code class="secret mono" bind:this={code}>{secret}</code>
    <div class="foot">
      <Button variant="print" size="sm" icon={copied ? 'check' : 'copy'} onclick={copy}>
        {copied ? '已复制' : '复制'}
      </Button>
      {#if failed}<span class="hint">无法写入剪贴板，已选中，请按 Ctrl / ⌘ + C</span>{/if}
      <span class="sr-only" role="status">{copied ? '已复制' : ''}</span>
    </div>
  </div>
</div>

<style>
  .ticket {
    --notch: 9px;
    display: flex;
    min-width: 0;
    border-radius: 2px;
    background: var(--film);
    color: var(--print);
    /* Semicircular punches at mid-height on both edges; the dialog surface shows through. */
    -webkit-mask:
      radial-gradient(circle var(--notch) at 0 50%, transparent 97%, #000) left / 50% 100% no-repeat,
      radial-gradient(circle var(--notch) at 100% 50%, transparent 97%, #000) right / 50% 100% no-repeat;
    mask:
      radial-gradient(circle var(--notch) at 0 50%, transparent 97%, #000) left / 50% 100% no-repeat,
      radial-gradient(circle var(--notch) at 100% 50%, transparent 97%, #000) right / 50% 100% no-repeat;
  }
  .stub {
    display: flex;
    flex: none;
    flex-direction: column;
    align-items: center;
    justify-content: space-between;
    gap: var(--s3);
    width: 52px;
    padding: var(--s4) 0;
    border-right: 1.5px dashed color-mix(in srgb, var(--edge-2) 45%, transparent);
  }
  .pb {
    font-size: var(--t-16);
    color: var(--edge);
  }
  .admit {
    color: var(--edge-2);
    writing-mode: vertical-rl;
    transform: rotate(180deg);
    letter-spacing: 0.12em;
  }
  .main {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: var(--s3);
    min-width: 0;
    padding: var(--s4) 24px var(--s4) var(--s4);
  }
  .head {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--s1) var(--s3);
  }
  .name {
    min-width: 0;
    font-size: var(--t-16);
    font-weight: 650;
    line-height: 1.3;
  }
  .scopes {
    color: var(--edge-2);
  }
  .secret {
    display: block;
    padding: var(--s3) 0;
    border-block: 1px solid var(--film-line);
    color: var(--edge);
    font-size: 15px;
    line-height: 1.6;
    letter-spacing: 0;
    overflow-wrap: anywhere;
    word-break: break-all;
    user-select: all;
    cursor: text;
  }
  .foot {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--s2) var(--s3);
  }
  .hint {
    color: var(--edge-2);
    font-size: var(--t-12);
  }

  @media (max-width: 480px) {
    .stub {
      width: 44px;
    }
    .main {
      padding-right: 20px;
    }
  }
</style>
