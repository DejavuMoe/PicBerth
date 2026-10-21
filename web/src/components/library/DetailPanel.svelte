<script lang="ts">
  import { rawUrl, thumbUrl } from '../../lib/api';
  import { facets } from '../../lib/facets.svelte';
  import { formatAbsolute, formatBytes } from '../../lib/format';
  import { daysLeft, formatName, hasAlpha, originalHref, sourceLabel } from '../../lib/image';
  import type { Library } from '../../lib/library.svelte';
  import type { Image } from '../../lib/types';
  import { LINK_FORMATS, setLinkFormat, uploads } from '../../lib/upload.svelte';
  import CopyButton from '../CopyButton.svelte';
  import Icon from '../Icon.svelte';
  import Seg from '../Seg.svelte';
  import Switch from '../Switch.svelte';
  import RenameField from './RenameField.svelte';
  import TagEditor from './TagEditor.svelte';

  interface Props {
    lib: Library;
    image: Image;
    onclose: () => void;
  }
  let { lib, image, onclose }: Props = $props();

  const trashed = $derived(image.state === 'trashed');
  const pub = $derived(image.visibility === 'public');
  const fmt = $derived(uploads.format);
  const fmtLabel = $derived(LINK_FORMATS.find((f) => f.value === fmt)?.label ?? '');
  const left = $derived(trashed ? daysLeft(image, lib.retention) : null);
  const knownTags = $derived((facets.active?.tags ?? []).map((t) => t.tag));
  // The thumbnail is a still: animated images and images without one show the original.
  const previewSrc = $derived(image.frames > 1 || !image.hasThumb ? rawUrl(image.id) : thumbUrl(image.id));

  let linkField = $state<HTMLInputElement | null>(null);
</script>

<div class="detail">
  <div class="detail-head">
    <h2 class="truncate" title={image.filename}>{image.filename}</h2>
    <button type="button" class="btn btn-ghost btn-icon btn-sm" aria-label="上一张" title="上一张 (←)" disabled={!lib.prev} onclick={() => lib.prev && lib.open(lib.prev.id)}>
      <Icon name="chevron-left" />
    </button>
    <button type="button" class="btn btn-ghost btn-icon btn-sm" aria-label="下一张" title="下一张 (→)" disabled={!lib.canNext} onclick={() => lib.step(1)}>
      <Icon name="chevron-right" />
    </button>
    <button type="button" class="btn btn-ghost btn-icon btn-sm" aria-label="关闭" title="关闭 (Esc)" onclick={onclose}>
      <Icon name="x" />
    </button>
  </div>

  <div class="detail-scroll">
    {#key image.id}
      <a
        class="preview"
        class:checker={hasAlpha(image)}
        href={originalHref(image)}
        target="_blank"
        rel="noreferrer"
        title="打开原图"
      >
        {#if image.broken}
          <Icon name="image" size={28} />
        {:else}
          <img src={previewSrc} alt={image.filename} />
        {/if}
      </a>
    {/key}

    {#if !trashed}
      <section>
        <div class="section-label">链接</div>
        <Seg
          label="链接格式"
          fill
          value={fmt}
          onchange={setLinkFormat}
          options={LINK_FORMATS.map((f) => ({ key: f.value, label: f.label }))}
        />
        <div class="link-box">
          <input bind:this={linkField} class="input" readonly value={image.links[fmt]} aria-label={`${fmtLabel} 链接`} onfocus={(e) => e.currentTarget.select()} />
          <CopyButton variant="primary" text={image.links[fmt]} field={linkField} />
        </div>
        {#if !pub}<p class="hint" style="margin-top: 6px">私有图片的链接只在登录后可以打开。</p>{/if}
      </section>

      <section>
        <div class="section-label">文件名</div>
        {#key image.id}
          <RenameField filename={image.filename} onsave={(n) => lib.rename(image.id, n)} />
        {/key}
      </section>

      <section>
        <div class="setting-row">
          <div>
            <div class="label">公开访问</div>
            <div class="hint">{pub ? '任何人都可以通过链接打开' : '只有登录后才能打开'}</div>
          </div>
          <Switch
            checked={pub}
            label="公开访问"
            disabled={lib.busyVisibility.has(image.id)}
            onchange={(v) => lib.setVisibility([image.id], v ? 'public' : 'private')}
          />
        </div>
      </section>

      <section>
        <div class="section-label">标签</div>
        {#key image.id}
          <TagEditor id={image.id} tags={image.tags} known={knownTags} onchange={(tags) => lib.setTags(image.id, tags)} />
        {/key}
      </section>
    {/if}

    <section>
      <div class="section-label">信息</div>
      <dl class="props">
        <dt>格式</dt>
        <dd>
          {formatName(image)}
          {#if image.frames > 1}<span class="badge">动图 · {image.frames} 帧</span>{/if}
        </dd>
        <dt>尺寸</dt>
        <dd class="num">{image.width} × {image.height}</dd>
        <dt>大小</dt>
        <dd>{formatBytes(image.size)}</dd>
        <dt>位置信息</dt>
        <dd>
          {#if image.hasGps}
            <span class="badge badge-warning"><Icon name="pin" size={12} />含位置</span>
          {:else}
            无
          {/if}
        </dd>
        <dt>来源</dt>
        <dd>{sourceLabel(image.source)}</dd>
        <dt>上传时间</dt>
        <dd class="num">{formatAbsolute(image.createdAt)}</dd>
        {#if trashed}
          <dt>删除时间</dt>
          <dd class="num">{image.trashedAt ? formatAbsolute(image.trashedAt) : '—'}</dd>
          {#if left}
            <dt>自动清除</dt>
            <dd>{left}</dd>
          {/if}
        {/if}
        <dt>ID</dt>
        <dd>
          <span class="mono truncate">{image.id}</span>
          <CopyButton iconOnly small variant="ghost" text={image.id} label="复制 ID" />
        </dd>
        <dt>SHA-256</dt>
        <dd>
          <span class="mono truncate" title={image.sha256}>{image.sha256}</span>
          <CopyButton iconOnly small variant="ghost" text={image.sha256} label="复制 SHA-256" />
        </dd>
      </dl>
    </section>
  </div>

  <div class="detail-foot">
    {#if trashed}
      <button type="button" class="btn btn-secondary" onclick={() => lib.restore([image.id])}>
        <Icon name="restore" />
        恢复
      </button>
      <span class="grow"></span>
      <button type="button" class="btn btn-danger" onclick={() => lib.requestPurge([image.id])}>永久删除</button>
    {:else}
      <a class="btn btn-secondary" href={rawUrl(image.id)} download={image.filename}>
        <Icon name="download" />
        下载
      </a>
      <span class="grow"></span>
      <button type="button" class="btn btn-danger" onclick={() => lib.trash([image.id])}>
        <Icon name="trash" />
        移到回收站
      </button>
    {/if}
  </div>
</div>
