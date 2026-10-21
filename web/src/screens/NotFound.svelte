<script lang="ts">
  import Button from '../components/Button.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import PageHeader from '../components/PageHeader.svelte';
</script>

<div class="page page-narrow">
  <PageHeader title="页面不存在" />
  <EmptyState message="这个地址没有对应的页面。">
    <Button href="/">回到上传</Button>
  </EmptyState>
</div>
