<script lang="ts">
  import { uploads } from '../lib/upload.svelte';
</script>

{#if uploads.drag.active}
  <div class="dropzone" aria-hidden="true">松开以上传</div>
{/if}
