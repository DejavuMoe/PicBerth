<script lang="ts">
  import Icon from '../components/Icon.svelte';
</script>

<div class="page" style="--page-w: 720px">
  <div class="empty">
    <span class="empty-icon"><Icon name="search" size={22} /></span>
    <strong>页面不存在</strong>
    <p>这个地址没有对应的页面。</p>
    <div class="actions">
      <a class="btn btn-secondary" href="/library">全部图片</a>
    </div>
  </div>
</div>
