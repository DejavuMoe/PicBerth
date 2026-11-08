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

function imageFormat(img: Pick<Image, 'mime'>): ImageFormat | null {
  return MIME_FORMAT[img.mime] ?? null;
}

export function formatName(img: Pick<Image, 'mime' | 'ext'>): string {
  const f = imageFormat(img);
  if (!f) return img.ext.toUpperCase();
  return { jpeg: 'JPEG', png: 'PNG', gif: 'GIF', webp: 'WebP', avif: 'AVIF' }[f];
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

/** "N 天后清除" for a trashed image, or null while the retention is unknown. */
export function daysLeft(img: Pick<Image, 'trashedAt'>, retentionDays: number | null, now = Date.now()): string | null {
  if (retentionDays === null || !img.trashedAt) return null;
  const left = Math.ceil((new Date(img.trashedAt).getTime() + retentionDays * 86_400_000 - now) / 86_400_000);
  return left <= 0 ? '即将清除' : `${left} 天后清除`;
}
