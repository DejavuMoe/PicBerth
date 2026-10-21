// Small derived facts about an Image used by several components.

import { rawUrl, thumbUrl } from './api';
import type { Image, ImageFormat } from './types';

const MIME_FORMAT: Record<string, ImageFormat> = {
  'image/jpeg': 'jpeg',
  'image/png': 'png',
  'image/gif': 'gif',
  'image/webp': 'webp',
  'image/avif': 'avif',
};

export function imageFormat(img: Pick<Image, 'mime'>): ImageFormat | null {
  return MIME_FORMAT[img.mime] ?? null;
}

/** Formats that can carry transparency are previewed on a checkerboard. */
export function hasAlpha(img: Pick<Image, 'mime'>): boolean {
  const f = imageFormat(img);
  return f === 'png' || f === 'webp' || f === 'gif' || f === 'avif';
}

export function formatName(img: Pick<Image, 'mime' | 'ext'>): string {
  const f = imageFormat(img);
  if (!f) return img.ext.toUpperCase();
  return { jpeg: 'JPEG', png: 'PNG', gif: 'GIF', webp: 'WebP', avif: 'AVIF' }[f];
}

export function isAnimated(img: Pick<Image, 'frames'>): boolean {
  return img.frames > 1;
}

export function sourceLabel(src: { kind: string; tokenName?: string }): string {
  if (src.kind === 'token') return src.tokenName ?? '';
  if (src.kind === 'cli') return '命令行';
  return '网页';
}

/** Label for a `source` filter key (`web`, `cli`, `token:<id>`). */
export function sourceKeyLabel(key: string, known: readonly { key: string; kind: string; tokenName?: string }[] = []): string {
  const hit = known.find((s) => s.key === key);
  if (hit) return sourceLabel(hit);
  if (key === 'web' || key === 'cli') return sourceLabel({ kind: key });
  return key;
}

export function thumbSrc(img: Pick<Image, 'id' | 'hasThumb'>): string {
  return img.hasThumb ? thumbUrl(img.id) : rawUrl(img.id);
}

/** The address "打开原图" opens: the public link when there is one, the signed-in raw route otherwise. */
export function originalHref(img: Pick<Image, 'id' | 'url' | 'visibility'>): string {
  return img.visibility === 'public' ? img.url : rawUrl(img.id);
}

/** "N 天后清除" for a trashed image, or null while the retention is unknown. */
export function daysLeft(img: Pick<Image, 'trashedAt'>, retentionDays: number | null, now = Date.now()): string | null {
  if (retentionDays === null || !img.trashedAt) return null;
  const left = Math.ceil((new Date(img.trashedAt).getTime() + retentionDays * 86_400_000 - now) / 86_400_000);
  return left <= 0 ? '即将清除' : `${left} 天后清除`;
}
