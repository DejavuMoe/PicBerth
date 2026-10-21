<script lang="ts">
  import Icon from '../Icon.svelte';

  let { filename, onsave }: { filename: string; onsave: (name: string) => Promise<boolean> } = $props();

  let edit = $state(false);
  let value = $state('');
  let busy = $state(false);

  const error = $derived(!value.trim() ? '请输入文件名' : value.length > 200 ? '文件名不能超过 200 个字符' : null);

  function begin() {
    value = filename;
    edit = true;
  }
  function cancel() {
    edit = false;
  }
  async function save() {
    if (error || busy) return;
    busy = true;
    const ok = await onsave(value.trim());
    busy = false;
    if (ok) edit = false;
  }
</script>

{#if !edit}
  <div class="setting-row">
    <span class="truncate" title={filename}>{filename}</span>
    <button type="button" class="btn btn-ghost btn-sm" onclick={begin}>
      <Icon name="edit" />
      重命名
    </button>
  </div>
{:else}
  <div class="rename">
    <!-- svelte-ignore a11y_autofocus -->
    <input
      class="input"
      autofocus
      bind:value
      aria-label="文件名"
      aria-invalid={!!error}
      onkeydown={(e) => {
        if (e.key === 'Enter') void save();
        if (e.key === 'Escape') {
          // Leave the editor without closing the drawer behind it.
          e.preventDefault();
          e.stopPropagation();
          cancel();
        }
      }}
    />
    <button type="button" class="btn btn-primary" disabled={!!error || busy} onclick={save}>保存</button>
    <button type="button" class="btn btn-ghost" onclick={cancel}>取消</button>
  </div>
  {#if error}
    <p class="error-text" style="margin-top: 6px">{error}</p>
  {:else}
    <p class="hint" style="margin-top: 6px">只修改显示的文件名，链接不变。</p>
  {/if}
{/if}
