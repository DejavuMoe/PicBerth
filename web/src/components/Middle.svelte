<script lang="ts">
  // Middle truncation: the head shrinks with an ellipsis, the tail (extension, last characters) stays.
  interface Props {
    text: string;
    tail?: number;
  }
  let { text, tail = 9 }: Props = $props();

  const parts = $derived.by(() => {
    const chars = [...text];
    if (chars.length <= tail + 6) return [text, ''] as const;
    return [chars.slice(0, chars.length - tail).join(''), chars.slice(chars.length - tail).join('')] as const;
  });
</script>

<span class="sr-only">{text}</span><span class="mid" title={text} aria-hidden="true"><span class="head">{parts[0]}</span><span class="tail">{parts[1]}</span></span>

<style>
  .mid {
    display: flex;
    min-width: 0;
    max-width: 100%;
    white-space: pre;
  }
  .head {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .tail {
    flex: none;
  }
</style>
