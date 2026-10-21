<script lang="ts">
  import type { Library, ViewMode } from '../../lib/library.svelte';
  import Seg from '../Seg.svelte';

  let { lib }: { lib: Library } = $props();
</script>

<Seg
  icons
  label="视图"
  value={lib.view}
  onchange={(v: ViewMode) => lib.setView(v)}
  options={[
    { key: 'table', icon: 'list', title: '列表' },
    { key: 'grid', icon: 'grid', title: '网格' },
  ]}
/>
