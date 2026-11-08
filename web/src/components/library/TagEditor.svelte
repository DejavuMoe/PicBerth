<script lang="ts">
  import Floating from '../Floating.svelte';
  import Icon from '../Icon.svelte';

  interface Props {
    tags: string[];
    /** Every tag in use on this side of the library, with how many images carry it. */
    all: { tag: string; count: number }[];
    onchange: (tags: string[]) => void;
  }
  let { tags, all, onchange }: Props = $props();

  // Same limits as the server: 20 tags per image, 32 characters each.
  const MAX_TAGS = 20;
  const MAX_LEN = 32;

  const uid = $props.id();
  const listId = `${uid}-tags`;
  let text = $state('');
  let open = $state(false);
  let input = $state<HTMLInputElement | null>(null);

  const clean = $derived(text.trim().replace(/^#/, ''));
  const suggestions = $derived(all.filter((t) => !tags.includes(t.tag) && t.tag.includes(clean)).slice(0, 6));
  const show = $derived(open && suggestions.length > 0);

  function add(value: string) {
    const t = value.trim().replace(/^#/, '').slice(0, MAX_LEN);
    if (!t || tags.includes(t) || tags.length >= MAX_TAGS) return;
    onchange([...tags, t]);
    text = '';
  }
</script>

{#each tags as t (t)}
  <span class="tag-edit">
    {t}
    <button type="button" aria-label={`移除标签 ${t}`} onclick={() => onchange(tags.filter((x) => x !== t))}>
      <Icon name="x" size="s" />
    </button>
  </span>
{/each}
{#if tags.length < MAX_TAGS}
  <input
    bind:this={input}
    class="tag-add"
    placeholder="添加"
    aria-label="添加标签"
    role="combobox"
    aria-expanded={show}
    aria-controls={show ? listId : undefined}
    maxlength={MAX_LEN + 1}
    autocomplete="off"
    spellcheck="false"
    value={text}
    onfocus={() => (open = true)}
    oninput={(e) => {
      text = e.currentTarget.value.replace(',', '');
      open = true;
    }}
    onkeydown={(e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        add(text);
      } else if (e.key === 'Backspace' && !text && tags.length) {
        onchange(tags.slice(0, -1));
      } else if (e.key === 'Escape') {
        e.stopPropagation();
        open = false;
      }
    }}
  />
{/if}
{#if show && input}
  <Floating anchor={input} class="menu" id={listId} role="listbox" aria-label="标签建议" onclose={() => (open = false)}>
    {#each suggestions as s (s.tag)}
      <button
        type="button"
        role="option"
        aria-selected="false"
        class="menu-item"
        onmousedown={(e) => e.preventDefault()}
        onclick={() => add(s.tag)}
      >
        {s.tag}
        <span class="end num">{s.count}</span>
      </button>
    {/each}
  </Floating>
{/if}
