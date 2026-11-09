import { test as base, expect, type Page } from '@playwright/test';
import { randomBytes } from 'node:crypto';
import { crc32, deflateSync } from 'node:zlib';
import type { Folder, ImagePage, ImageState } from '../../src/lib/types.js';

function chunk(type: string, data: Buffer): Buffer {
  const body = Buffer.concat([Buffer.from(type), data]);
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const checksum = Buffer.alloc(4);
  checksum.writeUInt32BE(crc32(body));
  return Buffer.concat([length, body, checksum]);
}

// Distinct pixels prevent owner-wide dedupe from reusing another test's image.
export function samplePng(): Buffer {
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(2, 0);
  ihdr.writeUInt32BE(2, 4);
  ihdr[8] = 8;
  ihdr[9] = 6;
  const raw = Buffer.concat([Buffer.from([0]), randomBytes(8), Buffer.from([0]), randomBytes(8)]);
  return Buffer.concat([
    Buffer.from('89504e470d0a1a0a', 'hex'),
    chunk('IHDR', ihdr), chunk('IDAT', deflateSync(raw)), chunk('IEND', Buffer.alloc(0)),
  ]);
}

export async function login(page: Page) {
  await page.goto('/login');
  await page.getByLabel('用户名').fill('testadmin');
  await page.getByLabel('密码').fill('TestPass123!');
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await expect(page).toHaveURL(/(?:\/|\/library)$/);
  await expect(page.locator('#main')).toBeVisible();
}

export async function checkPageHealth(page: Page) {
  await expect(page).toHaveTitle(/PicBerth$/);
  await expect(page.locator('#main')).toBeVisible();
  await expect(page.locator('.state[role="alert"], vite-error-overlay')).toHaveCount(0);
  await expect.poll(() => page.evaluate(
    () => document.documentElement.scrollWidth <= document.documentElement.clientWidth
  ), { message: 'Page should not overflow horizontally' }).toBe(true);
}

/** A row of the image table or a tile of the grid (not the upload tray). */
export function libraryItem(page: Page, filename: string) {
  return page.locator('table.table tbody tr, .grid .tile').filter({ hasText: filename });
}

/** A row of the upload tray. */
export function uploadRow(page: Page, filename: string) {
  return page.locator('.uploads .up-row').filter({ hasText: filename });
}

/** Waits until every named upload has finished (its row offers 查看), then closes the tray. */
export async function finishUploads(page: Page, filenames: string[]) {
  for (const name of filenames) {
    await expect(uploadRow(page, name).getByRole('button', { name: '查看', exact: true })).toBeVisible();
  }
  await page.locator('.uploads').getByRole('button', { name: '关闭', exact: true }).click();
  await expect(page.locator('.uploads')).toHaveCount(0);
}

export async function storedImages(page: Page, filename: string, state: ImageState = 'active') {
  const response = await page.request.get('/api/v1/images', { params: { q: filename, state } });
  expect(response.ok()).toBe(true);
  const result: ImagePage = await response.json();
  return result.items;
}

export async function storedFolders(page: Page): Promise<Folder[]> {
  const response = await page.request.get('/api/v1/folders');
  expect(response.ok()).toBe(true);
  return ((await response.json()) as { items: Folder[] }).items;
}

/** Creates a folder through the API so UI tests start from a known tree. */
export async function createFolderApi(page: Page, name: string, parentId: string | null = null): Promise<Folder> {
  const response = await page.request.post('/api/v1/folders', {
    data: { name, parentId },
    headers: { 'X-PicBerth-Client': 'web' },
  });
  expect(response.status(), await response.text()).toBe(201);
  return (await response.json()) as Folder;
}

export function uniqueName(prefix: string, testInfo: { project: { name: string } }): string {
  return `${prefix}-${testInfo.project.name}-${Date.now().toString(36)}`;
}

export const test = base.extend<{ pageErrors: void }>({
  pageErrors: [async ({ page }, use) => {
    const errors: string[] = [];
    page.on('pageerror', (error) => errors.push(error.message));
    await use();
    expect(errors, 'Unhandled page errors').toEqual([]);
  }, { auto: true }],
});

export { expect };
