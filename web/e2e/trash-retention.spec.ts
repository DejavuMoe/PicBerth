import type { Page } from '@playwright/test';
import {
  checkPageHealth, expect, finishUploads, libraryItem, login, samplePng, storedImages, test, uniqueName, useListView,
} from './harness/test-base.js';

async function settings(page: Page, patch: Record<string, number>) {
  const response = await page.request.patch('/api/v1/settings', {
    headers: { 'X-PicBerth-Client': 'web' }, data: patch,
  });
  expect(response.ok()).toBe(true);
  return response.json();
}

async function upload(page: Page, names: string[]) {
  await page.locator('input[type="file"]').setInputFiles(
    names.map((name) => ({ name, mimeType: 'image/png', buffer: samplePng() })),
  );
  await finishUploads(page, names);
}

async function select(page: Page, names: string[], mobile: boolean) {
  if (mobile) {
    await page.getByRole('button', { name: '更多', exact: true }).click();
    await page.getByRole('menuitem', { name: '选择', exact: true }).click();
  }
  for (const name of names) {
    const item = libraryItem(page, name);
    if (mobile) await item.click();
    else {
      await item.hover();
      await page.getByRole('checkbox', { name: `选择 ${name}`, exact: true }).click();
    }
  }
  return mobile ? page.getByRole('toolbar', { name: '批量操作' }) : page.locator('.head .bulk');
}

test('confirms zero-day deletion, supports retry, and saves other settings at zero', async ({ page }, info) => {
  const mobile = info.project.name === 'mobile';
  await page.emulateMedia({ colorScheme: 'dark', reducedMotion: 'no-preference' });
  await login(page);
  const before = await (await page.request.get('/api/v1/settings')).json();
  const prefix = uniqueName('zero', info);
  const names = ['a', 'b'].map((x) => `${prefix}-${x}.png`);
  let batchRequests = 0;
  try {
    await settings(page, { trashRetentionDays: 0 });
    await page.goto('/settings');
    await expect(page.locator('#s-trash')).toHaveValue('0');
    await expect(page.getByText('设为 0 天时，删除图片前需确认永久删除。')).toBeVisible();
    const audit = before.auditRetentionDays === 7 ? 8 : 7;
    await page.locator('#s-audit').fill(String(audit));
    await page.getByRole('button', { name: '保存', exact: true }).click();
    await expect(page.getByText('设置已保存')).toBeVisible();
    expect(await (await page.request.get('/api/v1/settings')).json()).toMatchObject({
      trashRetentionDays: 0, auditRetentionDays: audit,
    });

    await page.goto('/library');
    await upload(page, names);
    const dialog = page.getByRole('dialog', { name: /^永久删除/ });
    await page.route('**/api/v1/images/batch', async (route) => {
      if (++batchRequests === 1) {
        await route.fulfill({ status: 503, json: { code: 'unavailable' } });
      } else await route.continue();
    });
    await libraryItem(page, names[0]!).click();
    const single = page.locator('.detail-foot').getByRole('button', { name: '永久删除' });
    await single.click();
    const cancel = dialog.getByRole('button', { name: '取消' });
    await expect(cancel).toBeFocused();
    if (mobile) await cancel.tap();
    else {
      await page.keyboard.press('Tab');
      await expect(dialog.getByRole('button', { name: '永久删除' })).toBeFocused();
      await page.keyboard.press('Escape');
      await page.keyboard.press('Enter');
    }
    await expect(dialog).toHaveCount(0);
    expect(batchRequests).toBe(0);
    expect(await storedImages(page, names[0]!)).toHaveLength(1);

    await page.reload();
    const bulk = await select(page, names, mobile);
    await bulk.getByRole('button', { name: '永久删除' }).click();
    await expect(dialog).toHaveAccessibleName('永久删除 2 张图片？');
    const confirm = dialog.getByRole('button', { name: '永久删除' });
    await confirm.click();
    await expect(dialog.getByRole('alert')).toHaveText('永久删除失败，请重试。');
    for (const name of names) expect(await storedImages(page, name)).toHaveLength(1);
    await confirm.click();
    await expect(dialog).toHaveCount(0);
    await expect(page.getByText('已永久删除 2 张图片')).toBeVisible();
    await expect(page.getByRole('heading', { name: '图片', exact: true })).toBeFocused();
    await expect(page.locator('.toast').getByRole('button', { name: '撤销' })).toHaveCount(0);
    for (const name of names) {
      expect(await storedImages(page, name)).toEqual([]);
      expect(await storedImages(page, name, 'trashed')).toEqual([]);
    }
    expect(batchRequests).toBe(2);
    await checkPageHealth(page);
  } finally {
    await settings(page, { trashRetentionDays: before.trashRetentionDays, auditRetentionDays: before.auditRetentionDays });
  }
});

test('reconciles a confirmed deletion after the success response is lost', async ({ page }, info) => {
  await login(page);
  const before = await (await page.request.get('/api/v1/settings')).json();
  const name = `${uniqueName('retry', info)}.png`;
  try {
    await settings(page, { trashRetentionDays: 0 });
    await page.goto('/library');
    await upload(page, [name]);
    await libraryItem(page, name).click();
    await page.locator('.detail-foot').getByRole('button', { name: '永久删除' }).click();
    const dialog = page.getByRole('dialog', { name: /^永久删除/ });
    const confirm = dialog.getByRole('button', { name: '永久删除' });
    await page.route('**/api/v1/images/batch', async (route) => {
      const response = await route.fetch();
      expect((await response.json()).results).toMatchObject([{ ok: true }]);
      await route.fulfill({ status: 502, body: '' });
    }, { times: 1 });
    await confirm.click();
    await expect(dialog.getByRole('alert')).toHaveText('永久删除失败，请重试。');
    expect(await storedImages(page, name)).toEqual([]);
    expect(await storedImages(page, name, 'trashed')).toEqual([]);
    await confirm.click();
    await expect(dialog).toHaveCount(0);
    await expect(page.getByText('已永久删除', { exact: true })).toBeVisible();
    await expect(libraryItem(page, name)).toHaveCount(0);
  } finally {
    await settings(page, { trashRetentionDays: before.trashRetentionDays });
  }
});

test('blocks deletion until retention is known and rejects stale trash intent', async ({ page }, info) => {
  const mobile = info.project.name === 'mobile';
  await page.emulateMedia({ colorScheme: 'light' });
  await login(page);
  const before = await (await page.request.get('/api/v1/settings')).json();
  const name = `${uniqueName('policy', info)}.png`;
  try {
    await settings(page, { trashRetentionDays: 30 });
    await page.route('**/api/v1/settings', async (route) => {
      await route.fulfill({ status: 503, json: { code: 'unavailable' } });
    }, { times: 1 });
    await page.goto('/library');
    await upload(page, [name]);
    const bulk = await select(page, [name], mobile);
    await expect(bulk.getByRole('button', { name: '删除', exact: true })).toBeDisabled();
    await expect(page.getByText('无法读取回收站保留时间。')).toBeVisible();
    await page.getByRole('button', { name: '重试', exact: true }).click();
    const trash = bulk.getByRole('button', { name: '移到回收站' });
    await expect(trash).toBeEnabled();

    await settings(page, { trashRetentionDays: 0 });
    await trash.click();
    await expect(page.getByText('回收站保留时间已更改，请重新确认。')).toBeVisible();
    expect(await storedImages(page, name)).toHaveLength(1);
    expect(await storedImages(page, name, 'trashed')).toEqual([]);
    await bulk.getByRole('button', { name: '永久删除' }).click();
    const dialog = page.getByRole('dialog', { name: /^永久删除/ });
    await expect(dialog).toBeVisible();
    await settings(page, { trashRetentionDays: 30 });
    await dialog.getByRole('button', { name: '永久删除' }).click();
    await expect(dialog).toHaveCount(0);
    await expect(trash).toBeEnabled();
    expect(await storedImages(page, name)).toHaveLength(1);
    await trash.click();
    await page.locator('.toast').getByRole('button', { name: '撤销' }).click();
    await expect.poll(() => storedImages(page, name)).toHaveLength(1);

    await (await select(page, [name], mobile)).getByRole('button', { name: '移到回收站' }).click();
    await expect.poll(() => storedImages(page, name, 'trashed')).toHaveLength(1);
    await page.goto('/trash');
    await libraryItem(page, name).click();
    await page.locator('.detail-foot').getByRole('button', { name: '恢复', exact: true }).click();
    const undoRestore = page.locator('.toast').filter({ hasText: '已恢复' }).getByRole('button', { name: '撤销' });
    await expect(undoRestore).toBeVisible();
    await settings(page, { trashRetentionDays: 0 });
    await undoRestore.click();
    await expect(page.getByText('回收站保留时间已更改，请重新确认。')).toBeVisible();
    expect(await storedImages(page, name)).toHaveLength(1);
  } finally {
    await settings(page, { trashRetentionDays: before.trashRetentionDays });
  }
});

test('keeps the next row confirmation open when a previous trash purge finishes', async ({ page }, info) => {
  await useListView(page);
  await login(page);
  const prefix = uniqueName('purge', info);
  const a = `${prefix}-a.png`;
  const b = `${prefix}-b.png`;
  const before = await (await page.request.get('/api/v1/settings')).json();
  const release = Promise.withResolvers<void>();
  try {
    await settings(page, { trashRetentionDays: 30 });
    await page.goto('/library');
    await upload(page, [a, b]);
    const bulk = await select(page, [a, b], info.project.name === 'mobile');
    await bulk.getByRole('button', { name: '移到回收站' }).click();
    await expect(page.getByText('已将 2 张图片移到回收站')).toBeVisible();
    await page.goto('/trash');
    const dialog = page.getByRole('dialog', { name: /^永久删除/ });
    async function openPurge(name: string) {
      await page.getByRole('button', { name: `${name} 的更多操作` }).click();
      await page.getByRole('menuitem', { name: '永久删除' }).click();
      await expect(dialog).toHaveAccessibleName(`永久删除“${name}”？`);
    }
    await page.route('**/api/v1/images/batch', async (route) => {
      const response = await route.fetch();
      await release.promise;
      await route.fulfill({ response });
    }, { times: 1 });
    await openPurge(a);
    await dialog.getByRole('button', { name: '永久删除' }).click();
    await openPurge(b);
    await expect(dialog.getByRole('button', { name: '取消' })).toBeFocused();
    release.resolve();
    await expect(libraryItem(page, a)).toHaveCount(0);
    await expect(dialog).toHaveAccessibleName(`永久删除“${b}”？`);
    await expect(dialog.getByRole('button', { name: '取消' })).toBeFocused();
    await dialog.getByRole('button', { name: '取消' }).click();
    expect(await storedImages(page, b, 'trashed')).toHaveLength(1);
    await checkPageHealth(page);
  } finally {
    release.resolve();
    await settings(page, { trashRetentionDays: before.trashRetentionDays });
  }
});

test('retries only the unfinished selection after the second trash chunk loses its response', async ({ page }, info) => {
  await useListView(page);
  await login(page);
  const before = await (await page.request.get('/api/v1/settings')).json();
  const name = `${uniqueName('chunk', info)}.png`;
  try {
    await settings(page, { trashRetentionDays: 30 });
    await upload(page, [name]);
    const [template] = await storedImages(page, name);
    expect(template?.filename).toBe(name);
    const images = Array.from({ length: 503 }, (_, i) => ({ ...template, id: String(i), filename: `chunk-${i}.png`, broken: true }));
    const active = new Set(images.map((im) => im.id));
    let listReads = 0;
    let itemReads = 0;
    page.on('request', (req) => {
      if (/\/api\/v1\/images\/\d+$/.test(req.url())) itemReads++;
    });
    await page.route('**/api/v1/images?*', (route) => {
      listReads++;
      return route.fulfill({ json: { items: images.filter((im) => active.has(im.id)), nextCursor: null } });
    });
    const chunks: string[][] = [];
    await page.route('**/api/v1/images/batch', (route) => {
      const { action, ids, expectedTrashRetentionDays } = route.request().postDataJSON();
      if (action === 'restore') {
        expect(ids).toEqual(chunks[0]);
        for (const id of ids) active.add(id);
        return route.abort();
      }
      expect(expectedTrashRetentionDays).toBe(30);
      chunks.push(ids);
      if (chunks.length === 2) return route.abort();
      for (const id of ids) active.delete(id);
      return route.fulfill({ json: { results: ids.map((id: string) => ({ id, ok: true })) } });
    });
    await page.goto('/library');
    const bulk = await select(page, ['chunk-0.png'], info.project.name === 'mobile');
    await page.locator('.head .bulk').getByRole('checkbox', { name: '全选' }).click();
    await libraryItem(page, 'chunk-502.png').click();
    const trash = bulk.getByRole('button', { name: '移到回收站' });
    await trash.click();
    await expect(page.getByText('已选 2 张')).toBeVisible();
    await expect(libraryItem(page, 'chunk-502.png')).toHaveAttribute('aria-selected', 'false');
    await trash.click();
    await expect(page.getByText('已将 2 张图片移到回收站')).toBeVisible();
    expect(chunks.map((ids) => ids.length)).toEqual([500, 2, 2]);
    expect(chunks[2]).toEqual(['500', '501']);

    const beforeUndo = listReads;
    await page.locator('.toast').filter({ hasText: '已将 500 张图片移到回收站' }).getByRole('button', { name: '撤销' }).click();
    await expect(page.locator('table.table tbody tr')).toHaveCount(501);
    expect(listReads - beforeUndo).toBe(1);
    expect(itemReads).toBe(0);
  } finally {
    await settings(page, { trashRetentionDays: before.trashRetentionDays });
  }
});

test('undo from the trash reconciles a lost restore reply with the active screen', async ({ page }, info) => {
  await login(page);
  const before = await (await page.request.get('/api/v1/settings')).json();
  const name = `${uniqueName('undo', info)}.png`;
  try {
    await settings(page, { trashRetentionDays: 30 });
    await page.goto('/library');
    await upload(page, [name]);
    await libraryItem(page, name).click();
    await page.locator('.detail-foot').getByRole('button', { name: '移到回收站' }).click();
    const undo = page.locator('.toast').getByRole('button', { name: '撤销' });
    await expect(undo).toBeVisible();
    await page.locator('nav a[href="/trash"]:visible').click();
    await expect(libraryItem(page, name)).toBeVisible();
    await page.route('**/api/v1/images/batch', async (route) => {
      expect(route.request().postDataJSON().action).toBe('restore');
      expect((await (await route.fetch()).json()).results).toMatchObject([{ ok: true }]);
      await route.abort();
    }, { times: 1 });
    await undo.click();
    await expect(page.locator('.toast').getByText('已恢复', { exact: true })).toBeVisible();
    await expect(libraryItem(page, name)).toHaveCount(0);
    await expect(page.getByRole('heading', { name: '回收站', exact: true })).toBeFocused();
    expect(await storedImages(page, name)).toHaveLength(1);
    expect(await storedImages(page, name, 'trashed')).toEqual([]);
  } finally {
    await settings(page, { trashRetentionDays: before.trashRetentionDays });
  }
});
