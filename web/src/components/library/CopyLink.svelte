<script lang="ts">
  import { toast } from '../../lib/toast.svelte';
  import type { Image } from '../../lib/types';
  import { LINK_FORMATS, uploads } from '../../lib/upload.svelte';
  import CopyButton from '../CopyButton.svelte';

  let { image, variant = 'ghost' }: { image: Pick<Image, 'links'>; variant?: 'ghost' | 'overlay' } = $props();

  const label = $derived(LINK_FORMATS.find((f) => f.value === uploads.format)?.label ?? '');
</script>

<!-- Icon-only copy in the remembered link format. -->
<CopyButton
  iconOnly
  small
  {variant}
  text={image.links[uploads.format]}
  label={`复制 ${label}`}
  oncopied={() => toast(`已复制 ${label} 链接`)}
/>
