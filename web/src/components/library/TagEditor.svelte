<script lang="ts">
  import type { Facets } from '../../lib/types';
  import Combobox from '../Combobox.svelte';
  import Icon from '../Icon.svelte';

  interface Props {
    tags: string[];
    /** Tags in use elsewhere, with how many images carry each. */
    known: Facets['tags'];
    onchange: (tags: string[]) => void;
  }
  let { tags, known, onchange }: Props = $props();

  // Same limits as the server: 20 tags per image, 32 characters each.
  const MAX_TAGS = 20;
  const MAX_LEN = 32;

  const full = $derived(tags.length >= MAX_TAGS);
  const options = $derived(known.filter((t) => !tags.includes(t.tag)).map((t) => ({ value: t.tag, label: t.tag, count: t.count })));

  const normalize = (s: string) => s.trim().replace(/^#+/, '').trim().slice(0, MAX_LEN);
</script>

<div class="tag-editor">
  {#each tags as t (t)}
    <span class="tag tag-edit">
      {t}
      <button type="button" aria-label={`移除标签 ${t}`} onclick={() => onchange(tags.filter((x) => x !== t))}>
        <Icon name="x" size={12} />
      </button>
    </span>
  {/each}
  <Combobox
    label="添加标签"
    placeholder="添加标签"
    {options}
    {normalize}
    taken={(v) => tags.includes(v)}
    maxlength={MAX_LEN + 8}
    onpick={(v) => {
      if (!tags.includes(v) && tags.length < MAX_TAGS) onchange([...tags, v]);
    }}
    onemptybackspace={() => {
      if (tags.length) onchange(tags.slice(0, -1));
    }}
  >
    {#snippet trigger(open, toggle)}
      <button type="button" class="tag-add" aria-label="添加标签" aria-haspopup="listbox" aria-expanded={open} disabled={full} onclick={toggle}>
        <Icon name="plus" size={12} />
      </button>
    {/snippet}
  </Combobox>
</div>
