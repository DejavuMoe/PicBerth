<script lang="ts">
  import { onMount } from 'svelte';
  import CodeBlock from '../components/CodeBlock.svelte';
  import Icon from '../components/Icon.svelte';

  const SECTIONS = [
    { id: 'auth', label: '认证' },
    { id: 'upload', label: '上传' },
    { id: 'images', label: '图片' },
    { id: 'folders', label: '文件夹' },
    { id: 'errors', label: '错误' },
    { id: 'limits', label: '频率限制' },
    { id: 'clients', label: '客户端' },
  ];

  let scroller = $state<HTMLElement | null>(null);
  let current = $state(SECTIONS[0]!.id);

  // The contents mark the section being read.
  function mark() {
    if (!scroller) return;
    const top = scroller.getBoundingClientRect().top + 24;
    let cur = SECTIONS[0]!.id;
    for (const sec of SECTIONS) {
      const el = document.getElementById(sec.id);
      if (el && el.getBoundingClientRect().top <= top) cur = sec.id;
    }
    current = cur;
  }

  function jump(e: MouseEvent, id: string) {
    e.preventDefault();
    document.getElementById(id)?.scrollIntoView();
    history.replaceState(null, '', `#${id}`);
    current = id;
  }

  onMount(mark);

  const AUTH_CURL = String.raw`curl -H "Authorization: Bearer pb_…" \
  "https://pics.example.com/api/v1/images?limit=2"`;

  const UPLOAD_CURL = String.raw`curl -H "Authorization: Bearer pb_…" \
  -F file=@image.png \
  -F folder=k2m9q4xw8bd3f6hz \
  https://pics.example.com/api/v1/images`;

  const UPLOAD_RESPONSE = `HTTP/1.1 201 Created

{
  "id": "7k2m9q4xw8bd3f6hz1ca",
  "url": "https://img.example.com/i/7k2m9q4xw8bd3f6hz1ca.png",
  "filename": "image.png",
  "folderId": "k2m9q4xw8bd3f6hz",
  "size": 183224,
  "width": 1440,
  "height": 900,
  "duplicate": false,
  "links": {
    "url": "https://img.example.com/i/7k2m9q4xw8bd3f6hz1ca.png",
    "markdown": "![image](https://img.example.com/i/7k2m9q4xw8bd3f6hz1ca.png)",
    "html": "<img src=\\"https://img.example.com/i/7k2m9q4xw8bd3f6hz1ca.png\\" alt=\\"image\\" width=\\"1440\\" height=\\"900\\">",
    "bbcode": "[img]https://img.example.com/i/7k2m9q4xw8bd3f6hz1ca.png[/img]"
  }
}`;

  const PATCH_CURL = String.raw`curl -X PATCH \
  -H "Authorization: Bearer pb_…" \
  -H "Content-Type: application/json" \
  -d '{"folderId": "3n7vq2k9xwbd4f6h"}' \
  https://pics.example.com/api/v1/images/7k2m9q4xw8bd3f6hz1ca`;

  const BATCH_CURL = String.raw`curl -X POST \
  -H "Authorization: Bearer pb_…" \
  -H "Content-Type: application/json" \
  -d '{"action": "move", "ids": ["7k2m9q4xw8bd3f6hz1ca"], "folderId": null}' \
  https://pics.example.com/api/v1/images/batch`;

  const FOLDER_CREATE_CURL = String.raw`curl -X POST \
  -H "Authorization: Bearer pb_…" \
  -H "Content-Type: application/json" \
  -d '{"name": "云南", "parentId": "k2m9q4xw8bd3f6hz"}' \
  https://pics.example.com/api/v1/folders`;

  const FOLDER_CREATE_RESPONSE = `HTTP/1.1 201 Created

{
  "id": "3n7vq2k9xwbd4f6h",
  "name": "云南",
  "parentId": "k2m9q4xw8bd3f6hz",
  "depth": 3,
  "imageCount": 0,
  "createdAt": "2026-11-09T08:00:00Z"
}`;

  const ERROR_RESPONSE = `HTTP/1.1 409 Conflict
Content-Type: application/problem+json

{
  "type": "urn:picberth:problem:folder-not-empty",
  "title": "Folder not empty",
  "status": 409,
  "detail": "The folder contains images or folders.",
  "code": "folder-not-empty"
}`;

  const RATE_RESPONSE = `HTTP/1.1 429 Too Many Requests
Retry-After: 12
Content-Type: application/problem+json

{ "code": "rate-limited", "status": 429 }`;

  const CURL_CLIENT = String.raw`curl -H "Authorization: Bearer pb_…" \
  -F file=@image.png \
  -F folder=k2m9q4xw8bd3f6hz \
  https://pics.example.com/api/v1/images`;

  const PICGO = `API 地址: https://pics.example.com/api/v1/images
POST 参数名: file
JSON 路径: url
自定义请求头: {"Authorization":"Bearer pb_…"}
自定义 Body: {"folder":"k2m9q4xw8bd3f6hz"}`;

  const SHAREX = `{
  "Version": "14.1.0",
  "Name": "PicBerth",
  "DestinationType": "ImageUploader",
  "RequestMethod": "POST",
  "RequestURL": "https://pics.example.com/api/v1/images",
  "Headers": { "Authorization": "Bearer pb_…" },
  "Body": "MultipartFormData",
  "Arguments": { "folder": "k2m9q4xw8bd3f6hz" },
  "FileFormName": "file",
  "URL": "{json:url}",
  "ErrorMessage": "{json:detail}"
}`;
</script>

<svelte:head>
  <title>API 文档 - PicBerth</title>
</svelte:head>

{#snippet toc()}
  {#each SECTIONS as sec (sec.id)}
    <a
      class="nav-item"
      href={`#${sec.id}`}
      aria-current={current === sec.id ? 'true' : undefined}
      onclick={(e) => jump(e, sec.id)}>{sec.label}</a
    >
  {/each}
{/snippet}

<div class="shell doc-shell">
  <nav class="side" aria-label="目录">
    <div class="brand-row">
      <a class="brand" href="/library">
        <span class="brand-mark" aria-hidden="true"></span>
        <span class="brand-word">PicBerth</span>
      </a>
    </div>
    <div class="side-nav doc-toc">{@render toc()}</div>
  </nav>

  <main class="main">
    <div class="panel">
      <div class="panel-main">
        <div class="head">
          <div class="head-title">
            <h1>API 文档</h1>
          </div>
          <a class="btn btn-ghost" href="/tokens">令牌</a>
          <a class="btn btn-secondary" href="/api/v1/openapi.json">
            OpenAPI
            <Icon name="external" size="s" />
          </a>
        </div>
        <nav class="doc-tabs doc-toc" aria-label="目录">{@render toc()}</nav>
        <div class="scroll" bind:this={scroller} onscroll={mark}>
          <div class="doc-page">
    <section class="doc-sec" aria-label="地址">
      <dl class="dl">
        <dt>地址</dt>
        <dd class="mono">https://pics.example.com/api/v1</dd>
        <dt>格式</dt>
        <dd>JSON（UTF-8），上传使用 multipart/form-data</dd>
      </dl>
    </section>

    <section class="doc-sec" id="auth">
      <h2>认证</h2>
      <p>在请求头中带上令牌。令牌不能放在查询字符串里。</p>
      <CodeBlock text="Authorization: Bearer pb_…" />
      <div class="doc-scroll">
        <table class="doc-table">
          <thead>
            <tr><th>权限</th><th>允许</th></tr>
          </thead>
          <tbody>
            <tr><td>上传</td><td>上传图片；只能读取自己上传的图片</td></tr>
            <tr><td>管理</td><td>读取、上传、重命名、移动、删除图片；创建、重命名、移动、删除文件夹</td></tr>
          </tbody>
        </table>
      </div>
      <CodeBlock text={AUTH_CURL} />
    </section>

    <section class="doc-sec" id="upload">
      <h2>上传</h2>
      <div class="ep-list">
        <div class="ep">
          <span class="m">POST</span>
          <span class="p">/api/v1/images</span>
          <span class="sc">上传</span>
          <span class="d">上传一张图片</span>
        </div>
      </div>
      <div class="doc-scroll">
        <table class="doc-table">
          <thead>
            <tr><th>字段</th><th>类型</th><th>说明</th></tr>
          </thead>
          <tbody>
            <tr><td><code>file</code></td><td>文件</td><td>必填，图片文件</td></tr>
            <tr><td><code>folder</code></td><td>文本</td><td>文件夹 ID，见 <a href="#folders" onclick={(e) => jump(e, 'folders')}>文件夹</a>；省略时放在根目录；文件夹不存在时返回 <code>404</code></td></tr>
          </tbody>
        </table>
      </div>
      <CodeBlock text={UPLOAD_CURL} />
      <CodeBlock text={UPLOAD_RESPONSE} />
      <p>已存在相同的图片时返回 <code>200</code> 和 <code>"duplicate": true</code>，链接不变。</p>
    </section>

    <section class="doc-sec" id="images">
      <h2>图片</h2>
      <div class="ep-list">
        <div class="ep">
          <span class="m">GET</span>
          <span class="p">/api/v1/images</span>
          <span class="sc">管理</span>
          <span class="d">列出图片</span>
        </div>
        <div class="ep">
          <span class="m">GET</span>
          <span class="p">/api/v1/images/{'{id}'}</span>
          <span class="sc">管理</span>
          <span class="d">获取图片</span>
        </div>
        <div class="ep">
          <span class="m">PATCH</span>
          <span class="p">/api/v1/images/{'{id}'}</span>
          <span class="sc">管理</span>
          <span class="d">重命名或移动图片</span>
        </div>
        <div class="ep">
          <span class="m">DELETE</span>
          <span class="p">/api/v1/images/{'{id}'}</span>
          <span class="sc">管理</span>
          <span class="d">移到回收站</span>
        </div>
        <div class="ep">
          <span class="m">POST</span>
          <span class="p">/api/v1/images/batch</span>
          <span class="sc">管理</span>
          <span class="d">批量移动、移到回收站、恢复，最多 500 张</span>
        </div>
      </div>
      <div class="doc-scroll">
        <table class="doc-table">
          <thead>
            <tr><th>查询参数</th><th>取值</th></tr>
          </thead>
          <tbody>
            <tr><td><code>folder</code></td><td><code>root</code> 或文件夹 ID；省略时包含全部文件夹</td></tr>
            <tr><td><code>recursive</code></td><td><code>true</code> 时包含子文件夹</td></tr>
            <tr><td><code>q</code></td><td>文件名、ID 或 SHA-256 前缀</td></tr>
            <tr><td><code>format</code></td><td><code>jpeg</code> <code>png</code> <code>gif</code> <code>webp</code> <code>avif</code>，可重复</td></tr>
            <tr><td><code>source</code></td><td><code>web</code> <code>cli</code> <code>token:&lt;id&gt;</code>，可重复</td></tr>
            <tr><td><code>since</code> <code>until</code></td><td>RFC 3339 时间</td></tr>
            <tr><td><code>sort</code></td><td><code>new</code> <code>old</code> <code>large</code> <code>small</code> <code>name</code></td></tr>
            <tr><td><code>limit</code> <code>cursor</code></td><td>1 到 200，默认 60；<code>nextCursor</code> 翻页</td></tr>
          </tbody>
        </table>
      </div>
      <CodeBlock text={PATCH_CURL} />
      <CodeBlock text={BATCH_CURL} />
    </section>

    <section class="doc-sec" id="folders">
      <h2>文件夹</h2>
      <div class="ep-list">
        <div class="ep">
          <span class="m">GET</span>
          <span class="p">/api/v1/folders</span>
          <span class="sc">上传</span>
          <span class="d">列出全部文件夹</span>
        </div>
        <div class="ep">
          <span class="m">POST</span>
          <span class="p">/api/v1/folders</span>
          <span class="sc">管理</span>
          <span class="d">创建文件夹</span>
        </div>
        <div class="ep">
          <span class="m">PATCH</span>
          <span class="p">/api/v1/folders/{'{id}'}</span>
          <span class="sc">管理</span>
          <span class="d">重命名或移动文件夹</span>
        </div>
        <div class="ep">
          <span class="m">DELETE</span>
          <span class="p">/api/v1/folders/{'{id}'}</span>
          <span class="sc">管理</span>
          <span class="d">删除空文件夹</span>
        </div>
      </div>
      <dl class="dl">
        <dt>层数</dt>
        <dd>最多 3 层，例如 <code>旅行/2026/云南</code></dd>
        <dt>名称</dt>
        <dd>1 到 64 个字符，不含 <code>/</code> 和 <code>\</code>，同一层不能重名</dd>
        <dt>删除</dt>
        <dd>文件夹里有图片或子文件夹时返回 <code>folder-not-empty</code></dd>
        <dt>链接</dt>
        <dd>移动图片和文件夹不会改变图片链接</dd>
      </dl>
      <CodeBlock text={FOLDER_CREATE_CURL} />
      <CodeBlock text={FOLDER_CREATE_RESPONSE} />
    </section>

    <section class="doc-sec" id="errors">
      <h2>错误</h2>
      <CodeBlock text={ERROR_RESPONSE} />
      <div class="doc-scroll">
        <table class="doc-table">
          <thead>
            <tr><th>code</th><th class="r">状态</th><th>含义</th></tr>
          </thead>
          <tbody>
            <tr><td><code>validation</code></td><td class="r num">400</td><td>请求或字段无效</td></tr>
            <tr><td><code>unauthorized</code></td><td class="r num">401</td><td>令牌缺失、无效或已吊销</td></tr>
            <tr><td><code>forbidden</code></td><td class="r num">403</td><td>令牌权限不足</td></tr>
            <tr><td><code>not-found</code></td><td class="r num">404</td><td>图片或文件夹不存在</td></tr>
            <tr><td><code>conflict</code></td><td class="r num">409</td><td>当前状态不允许该操作</td></tr>
            <tr><td><code>folder-exists</code></td><td class="r num">409</td><td>同一层已有同名文件夹</td></tr>
            <tr><td><code>folder-not-empty</code></td><td class="r num">409</td><td>文件夹不为空</td></tr>
            <tr><td><code>folder-depth</code></td><td class="r num">422</td><td>超过 3 层</td></tr>
            <tr><td><code>too-large</code></td><td class="r num">413</td><td>超过单文件上限</td></tr>
            <tr><td><code>unsupported-format</code></td><td class="r num">415</td><td>不支持的格式</td></tr>
            <tr><td><code>invalid-image</code></td><td class="r num">422</td><td>图片已损坏</td></tr>
            <tr><td><code>rate-limited</code></td><td class="r num">429</td><td>超过频率上限</td></tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="doc-sec" id="limits">
      <h2>频率限制</h2>
      <p>上传每分钟最多 60 次。超出时返回 <code>429</code>，<code>Retry-After</code> 为等待的秒数。</p>
      <CodeBlock text={RATE_RESPONSE} />
    </section>

    <section class="doc-sec" id="clients">
      <h2>客户端</h2>
      <h3>curl</h3>
      <CodeBlock text={CURL_CLIENT} />
      <h3>PicGo</h3>
      <CodeBlock text={PICGO} />
      <h3>ShareX</h3>
      <CodeBlock text={SHAREX} />
    </section>
          </div>
        </div>
      </div>
    </div>
  </main>
</div>
