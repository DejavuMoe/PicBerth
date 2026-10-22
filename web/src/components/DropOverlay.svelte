<script lang="ts">
  import { uploads } from '../lib/upload.svelte';
  import Icon from './Icon.svelte';
</script>

{#if uploads.drag.active}
  <div class="drop-overlay" aria-hidden="true">
    <div>
      <span class="dz-icon"><Icon name="upload" size={22} /></span>
      <strong>松开以上传</strong>
    </div>
  </div>
{/if}
