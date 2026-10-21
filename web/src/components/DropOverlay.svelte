<script lang="ts">
  import { uploads } from '../lib/upload.svelte';
  import Icon from './Icon.svelte';
</script>

{#if uploads.drag.active}
  <div class="drop-overlay" aria-hidden="true">
    <div>
      <Icon name="upload" size={28} />
      <strong>松开以上传</strong>
      <span class="hint">可以一次放入多张图片</span>
    </div>
  </div>
{/if}
