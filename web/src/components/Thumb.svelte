<script lang="ts">
  import { rawUrl } from '../lib/api';
  import { thumbSrc } from '../lib/image';
  import type { Image } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    /** A library image, or a bare `src` (upload previews). */
    image?: Pick<Image, 'id' | 'hasThumb' | 'mime' | 'broken'>;
    src?: string;
    /** 28px, for list rows. */
    size?: 'sm';
  }
  let { image, src, size }: Props = $props();

  const url = $derived(src ?? (image ? thumbSrc(image) : ''));
  // Without a thumbnail the server answers 404; fall back to the original once, then to a placeholder.
  let step = $state(0);
  $effect(() => {
    url;
    step = 0;
  });
  const shown = $derived(step === 0 ? url : step === 1 && image ? rawUrl(image.id) : '');
</script>

<span class="thumb" class:thumb-sm={size === 'sm'}>
  {#if shown && !image?.broken}
    <img src={shown} alt="" loading="lazy" draggable="false" onerror={() => (step += 1)} />
  {:else}
    <Icon name="image" />
  {/if}
</span>
