<script lang="ts">
  import Icon from '../Icon.svelte';
  import Modal from '../Modal.svelte';
  import Seg from '../Seg.svelte';
  import { copyText } from '../../lib/clipboard';
  import { toast } from '../../lib/toast.svelte';

  let { secret, ondone }: { secret: string; ondone: () => void } = $props();

  type Client = 'picgo' | 'sharex' | 'curl';
  const CLIENTS: { key: Client; label: string }[] = [
    { key: 'picgo', label: 'PicGo' },
    { key: 'sharex', label: 'ShareX' },
    { key: 'curl', label: 'curl' },
  ];

  let client = $state<Client>('picgo');

  // Examples use the origin the page is served from.
  const endpoint = `${location.origin}/api/v1/images`;
  const snippets = $derived.by(() => ({
    picgo: [
      `API 地址: ${endpoint}`,
      'POST 参数名: file',
      'JSON 路径: url',
      `自定义请求头: ${JSON.stringify({ Authorization: `Bearer ${secret}` })}`,
    ].join('\n'),
    sharex: [`RequestURL: ${endpoint}`, 'FileFormName: file', `Headers: Authorization: Bearer ${secret}`, 'URL: {json:url}'].join('\n'),
    curl: `curl -H "Authorization: Bearer ${secret}" \\\n  -F file=@image.png \\\n  ${endpoint}`,
  }));

  async function copySnippet() {
    const r = await copyText(snippets[client]);
    if (r === 'failed') return toast('当前连接不安全，无法写入剪贴板');
    toast('已复制');
  }

  function downloadSxcu() {
    const config = {
      Version: '15.0.0',
      Name: `PicBerth (${location.host})`,
      DestinationType: 'ImageUploader',
      RequestMethod: 'POST',
      RequestURL: endpoint,
      Headers: { Authorization: `Bearer ${secret}` },
      Body: 'MultipartFormData',
      FileFormName: 'file',
      URL: '{json:url}',
      ErrorMessage: '{json:detail}',
    };
    const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
    const href = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = href;
    a.download = 'PicBerth.sxcu';
    document.body.append(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(href), 1000);
    toast('已下载 PicBerth.sxcu');
  }
</script>

<Modal title="令牌已创建" wide dismissible={false} onclose={ondone}>
  <div class="notice">
    <Icon name="alert" />
    令牌只显示这一次
  </div>
  <div class="secret">
    <span class="mono">{secret}</span>
    <button
      type="button"
      class="btn btn-ghost btn-sm"
      onclick={() =>
        void copyText(secret).then((r) => {
          if (r === 'failed') toast('当前连接不安全，无法写入剪贴板');
          else toast('已复制令牌');
        })}
    >
      <Icon name="copy" size="s" />
      复制
    </button>
  </div>
  <div class="field">
    <div class="row" style="justify-content: space-between;">
      <Seg label="客户端" value={client} onchange={(k) => (client = k)} options={CLIENTS} />
      {#if client === 'sharex'}
        <button type="button" class="btn btn-ghost btn-sm" onclick={downloadSxcu}>
          <Icon name="download" size="s" />
          下载 .sxcu
        </button>
      {:else}
        <button type="button" class="btn btn-ghost btn-sm" onclick={() => void copySnippet()}>
          <Icon name="copy" size="s" />
          复制
        </button>
      {/if}
    </div>
    <pre class="snippet">{snippets[client]}</pre>
  </div>
  {#snippet footer()}
    <button type="button" class="btn btn-primary" data-autofocus onclick={ondone}>完成</button>
  {/snippet}
</Modal>
