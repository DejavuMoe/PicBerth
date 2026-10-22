<script lang="ts">
  import Icon from './Icon.svelte';

  /** `large`: the 32px mark above the sign-in and setup forms. */
  let { large = false }: { large?: boolean } = $props();
</script>

<span class="brand-mark" class:brand-mark-lg={large}><Icon name="image" size={large ? 18 : 12} /></span>
