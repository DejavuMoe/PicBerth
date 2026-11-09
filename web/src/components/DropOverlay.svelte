<script lang="ts">
  import { uploads, uploadTarget } from '../lib/upload.svelte';

  const destination = $derived(uploadTarget.current.id ? uploadTarget.current.label : '');
</script>

{#if uploads.drag.active}
  <div class="dropzone" aria-hidden="true">{destination ? `松开以上传到 ${destination}` : '松开以上传'}</div>
{/if}
