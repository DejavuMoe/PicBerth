import type { Page } from '@playwright/test';
import {
  checkPageHealth, expect, finishUploads, libraryItem, login, samplePng, storedImages, test, uniqueName,
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

test('blocks deletion until retention is known and rejects stale trash intent', async ({ page }, info) => {
  const mobile = info.project.name === 'mobile';
  await page.emulateMedia({ colorScheme: 'light' });
  await login(page);
  const before = await (await page.request.get('/api/v1/settings')).json();
  const name = `policy-${info.project.name}.png`;
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
