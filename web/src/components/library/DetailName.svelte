<script lang="ts">
  import Icon from '../Icon.svelte';

  interface Props {
    filename: string;
    /** Trashed images keep their name. */
    editable: boolean;
    /** Resolves to whether the new name was saved. */
    onsave: (name: string) => Promise<boolean>;
  }
  let { filename, editable, onsave }: Props = $props();

  let edit = $state(false);
  let value = $state('');
  let busy = $state(false);
  let input = $state<HTMLInputElement | null>(null);

  const error = $derived(!value.trim() ? '请输入文件名' : value.length > 200 ? '文件名不能超过 200 个字符' : null);

  // Entering edit mode focuses the field with the name selected, without its extension.
  $effect(() => {
    if (!edit || !input) return;
    input.focus();
    const dot = input.value.lastIndexOf('.');
    input.setSelectionRange(0, dot > 0 ? dot : input.value.length);
  });

  function begin() {
    value = filename;
    edit = true;
  }
  function cancel() {
    edit = false;
  }
  async function save() {
    if (error || busy) return;
    if (value.trim() === filename) return cancel();
    busy = true;
    const ok = await onsave(value.trim());
    busy = false;
    if (ok) edit = false;
  }
</script>

{#if !edit}
  <h2 class="d-name">
    {#if editable}
      <button type="button" class="d-name-btn" title="重命名" aria-label={`重命名 ${filename}`} onclick={begin}>
        <span>{filename}</span>
        <Icon name="edit" size={14} />
      </button>
    {:else}
      <span>{filename}</span>
    {/if}
  </h2>
{:else}
  <div class="d-rename">
    <div class="rename">
      <input
        bind:this={input}
        class="input"
        bind:value
        aria-label="文件名"
        aria-invalid={!!error}
        onkeydown={(e) => {
          if (e.key === 'Enter') void save();
          if (e.key === 'Escape') {
            // Leave the editor without closing the detail behind it.
            e.preventDefault();
            e.stopPropagation();
            cancel();
          }
        }}
      />
      <button type="button" class="btn btn-secondary" disabled={!!error || busy} onclick={save}>保存</button>
      <button type="button" class="btn btn-ghost" onclick={cancel}>取消</button>
    </div>
    {#if error}<p class="error-text">{error}</p>{/if}
  </div>
{/if}
