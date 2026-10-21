<script lang="ts">
  import Icon from '../Icon.svelte';

  interface Props {
    id: string;
    tags: string[];
    known: string[];
    onchange: (tags: string[]) => void;
  }
  let { id, tags, known, onchange }: Props = $props();

  let value = $state('');

  function add() {
    const t = value.trim();
    if (t && !tags.includes(t)) onchange([...tags, t]);
    value = '';
  }
</script>

<div class="tag-editor">
  {#each tags as t (t)}
    <span class="tag">
      {t}
      <button type="button" aria-label={`移除标签 ${t}`} onclick={() => onchange(tags.filter((x) => x !== t))}>
        <Icon name="x" size={12} />
      </button>
    </span>
  {/each}
  <input
    class="input"
    placeholder="添加标签"
    aria-label="添加标签"
    list={`tags-${id}`}
    maxlength={20}
    bind:value
    onkeydown={(e) => {
      if (e.key === 'Enter' || e.key === ',') {
        e.preventDefault();
        add();
      }
      if (e.key === 'Backspace' && !value && tags.length) onchange(tags.slice(0, -1));
    }}
    onblur={add}
  />
  <datalist id={`tags-${id}`}>
    {#each known.filter((t) => !tags.includes(t)) as t (t)}
      <option value={t}></option>
    {/each}
  </datalist>
</div>
