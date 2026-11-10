<script lang="ts">
  import { uploads, uploadTarget } from '../lib/upload.svelte';

  const destination = $derived(uploadTarget.current.id ? uploadTarget.current.label : '');
</script>

{#if uploads.drag.active}
  <div class="dropzone" aria-hidden="true">
    <svg class="frame" aria-hidden="true"><rect x="0" y="0" width="100%" height="100%" rx="12" /></svg>
    <span class="drop-label">
      <span class="medium">松开以上传</span>
      {#if destination}<span class="where">{destination}</span>{/if}
    </span>
  </div>
{/if}
