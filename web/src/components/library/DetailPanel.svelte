<script lang="ts">
  import { rawUrl, thumbUrl } from '../../lib/api';
  import { facets } from '../../lib/facets.svelte';
  import { formatAbsolute, formatBytes, formatRelative } from '../../lib/format';
  import { daysLeft, formatName, hasAlpha, originalHref, sourceLabel } from '../../lib/image';
  import { downloadImage, type Library } from '../../lib/library.svelte';
  import type { Image, LinkFormat } from '../../lib/types';
  import { LINK_FORMATS, setLinkFormat, uploads } from '../../lib/upload.svelte';
  import CopyButton from '../CopyButton.svelte';
  import Icon from '../Icon.svelte';
  import Switch from '../Switch.svelte';
  import Tabs from '../Tabs.svelte';
  import DetailName from './DetailName.svelte';
  import TagEditor from './TagEditor.svelte';

  interface Props {
    lib: Library;
    image: Image;
    /** Docked column beside the list, right drawer, or full phone sheet. */
    mode: 'dock' | 'drawer' | 'sheet';
    /** In a drawer or sheet this is the overlay's animated close. */
    onclose: () => void;
  }
  let { lib, image, mode, onclose }: Props = $props();

  const trashed = $derived(image.state === 'trashed');
  const pub = $derived(image.visibility === 'public');
  const fmt = $derived(uploads.format);
  const fmtLabel = $derived(LINK_FORMATS.find((f) => f.value === fmt)?.label ?? '');
  const left = $derived(trashed ? daysLeft(image, lib.retention) : null);
  const knownTags = $derived(facets.active?.tags ?? []);
  // The thumbnail is a still: animated images and images without one show the original.
  const previewSrc = $derived(image.frames > 1 || !image.hasThumb ? rawUrl(image.id) : thumbUrl(image.id));

  let linkField = $state<HTMLInputElement | null>(null);
</script>

<div class="detail detail-{mode}">
  <div class="d-top">
    <button type="button" class="btn btn-ghost btn-icon" aria-label="上一张" title="上一张 (←)" disabled={!lib.prev} onclick={() => lib.prev && lib.open(lib.prev.id)}>
      <Icon name="chevron-left" />
    </button>
    <button type="button" class="btn btn-ghost btn-icon" aria-label="下一张" title="下一张 (→)" disabled={!lib.canNext} onclick={() => lib.step(1)}>
      <Icon name="chevron-right" />
    </button>
    <span class="sp"></span>
    <button type="button" class="btn btn-ghost btn-icon" aria-label="关闭" title="关闭 (Esc)" onclick={onclose}>
      <Icon name="x" />
    </button>
  </div>

  <div class="detail-scroll">
    <div class="preview" class:checker={hasAlpha(image)}>
      <a class="preview-img" href={originalHref(image)} target="_blank" rel="noreferrer" aria-label="打开原图" tabindex="-1">
        {#key image.id}
          {#if image.broken}
            <Icon name="image" size={28} />
          {:else}
            <img src={previewSrc} alt={image.filename} />
          {/if}
        {/key}
      </a>
      <a class="preview-open" href={originalHref(image)} target="_blank" rel="noreferrer">
        <Icon name="external" size={14} />
        打开原图
      </a>
    </div>

    <div class="d-body">
      <div>
        {#key image.id}
          <DetailName filename={image.filename} editable={!trashed} onsave={(n) => lib.rename(image.id, n)} />
        {/key}
        <div class="d-meta num">
          <span>{formatName(image)}</span>
          <span>{image.width} × {image.height}</span>
          <span>{formatBytes(image.size)}</span>
          {#if image.frames > 1}<span>动图 {image.frames} 帧</span>{/if}
          {#if left}
            <span class="badge">{left}</span>
          {:else}
            <time title={formatAbsolute(image.createdAt)}>{formatRelative(image.createdAt)}</time>
          {/if}
        </div>
      </div>

      {#if !trashed}
        <div class="d-link">
          <Tabs
            label="链接格式"
            value={fmt}
            onchange={(v: LinkFormat) => setLinkFormat(v)}
            options={LINK_FORMATS.map((f) => ({ key: f.value, label: f.label }))}
          />
          <div class="link-box">
            <input bind:this={linkField} class="input mono" readonly value={image.links[fmt]} aria-label={`${fmtLabel} 链接`} onfocus={(e) => e.currentTarget.select()} />
            <CopyButton iconOnly variant="ghost" text={image.links[fmt]} field={linkField} label={`复制 ${fmtLabel}`} />
          </div>
        </div>
      {/if}

      <dl class="kv">
        {#if !trashed}
          <dt>公开</dt>
          <dd>
            <Switch
              checked={pub}
              label="公开"
              disabled={lib.busyVisibility.has(image.id)}
              onchange={(v) => lib.setVisibility([image.id], v ? 'public' : 'private')}
            />
          </dd>
          <dt class="top">标签</dt>
          <dd class="top">
            {#key image.id}
              <TagEditor tags={image.tags} known={knownTags} onchange={(tags) => lib.setTags(image.id, tags)} />
            {/key}
          </dd>
        {/if}
        <dt>来源</dt>
        <dd>{sourceLabel(image.source)}</dd>
        <dt>上传时间</dt>
        <dd class="num">{formatAbsolute(image.createdAt)}</dd>
        {#if trashed}
          <dt>删除时间</dt>
          <dd class="num">{image.trashedAt ? formatAbsolute(image.trashedAt) : '—'}</dd>
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
    </div>
  </div>

  <div class="detail-foot">
    {#if trashed}
      <button type="button" class="btn btn-ghost" onclick={() => lib.restore([image.id])}>
        <Icon name="restore" />
        恢复
      </button>
      <button type="button" class="btn btn-danger-ghost" onclick={() => lib.requestPurge([image.id])}>永久删除</button>
    {:else}
      <button type="button" class="btn btn-ghost" onclick={() => downloadImage(image)}>
        <Icon name="download" />
        下载
      </button>
      <button type="button" class="btn btn-danger-ghost" onclick={() => lib.trash([image.id])}>
        <Icon name="trash" />
        移到回收站
      </button>
    {/if}
  </div>
</div>
