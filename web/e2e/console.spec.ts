import {
  checkPageHealth, createFolderApi, expect, finishUploads, libraryItem, login, samplePng, test, uniqueName,
} from './harness/test-base.js';

test('has no tag or visibility controls anywhere in the library', async ({ page }, testInfo) => {
  const mobile = testInfo.project.name === 'mobile';
  await login(page);
  const name = uniqueName('plain', testInfo) + '.png';
  await page.locator('input[type="file"]').setInputFiles({ name, mimeType: 'image/png', buffer: samplePng() });
  await finishUploads(page, [name]);
  await expect(libraryItem(page, name)).toBeVisible();

  if (!mobile) {
    await expect(page.getByRole('columnheader', { name: '标签' })).toHaveCount(0);
  }
  await expect(page.locator('.tag')).toHaveCount(0);

  await page.getByRole('button', { name: '筛选', exact: true }).click();
  await expect(page.locator('.fp-label', { hasText: '格式' })).toBeVisible();
  await expect(page.getByText('标签', { exact: true })).toHaveCount(0);
  await page.keyboard.press('Escape');

  await libraryItem(page, name).click();
  const detail = page.locator('.detail');
  await expect(detail).toBeVisible();
  await expect(detail.locator('.props dt').first()).toHaveText('文件夹');
  await expect(detail.getByText('图片', { exact: true }).first()).toBeVisible();
  await expect(detail.getByText('标签', { exact: true })).toHaveCount(0);
  await expect(detail.getByText('公开', { exact: true })).toHaveCount(0);
  await expect(detail.getByText('私有', { exact: true })).toHaveCount(0);
  await page.keyboard.press('Escape');
  await checkPageHealth(page);
});

test('offers only the upload and manage token scopes and links the API reference', async ({ page }) => {
  await login(page);
  await page.goto('/tokens');
  const docs = page.getByRole('link', { name: /API 文档/ });
  await expect(docs).toHaveAttribute('href', '/docs/api');
  await expect(docs).toHaveAttribute('target', '_blank');

  await page.getByRole('button', { name: '新建令牌' }).click();
  await expect(page.getByRole('checkbox', { name: '上传', exact: true })).toBeVisible();
  await expect(page.getByRole('checkbox', { name: '管理', exact: true })).toBeVisible();
  await expect(page.getByRole('checkbox', { name: '只读', exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: '取消', exact: true }).click();
  await checkPageHealth(page);
});

test('serves the API reference as a page of its own', async ({ page }) => {
  await page.goto('/docs/api');
  await expect(page.getByRole('heading', { level: 1, name: 'API 文档' })).toBeVisible();
  await expect(page.getByRole('link', { name: /OpenAPI/ })).toHaveAttribute('href', '/api/v1/openapi.json');
  await expect(page.getByRole('heading', { level: 2, name: '文件夹' })).toBeVisible();
  await expect(page.getByText('最多 3 层，例如')).toBeVisible();
  await expect(page.getByText('上传每分钟最多 60 次。超出时返回 429，Retry-After 为等待的秒数。')).toBeVisible();
});

test('settings keeps only the five editable values and validates the audit retention', async ({ page }) => {
  await login(page);
  await page.goto('/settings');
  await expect(page.locator('#s-max')).toBeVisible();
  await expect(page.locator('#s-trash')).toBeVisible();
  await expect(page.locator('#s-audit')).toBeVisible();
  await expect(page.locator('#s-px')).toHaveCount(0);
  await expect(page.locator('#s-rate')).toHaveCount(0);
  await expect(page.locator('#s-idle')).toHaveCount(0);
  await expect(page.locator('#s-abs')).toHaveCount(0);
  await expect(page.getByRole('heading', { level: 2, name: '操作记录' })).toBeVisible();

  const audit = page.locator('#s-audit');
  await audit.fill('6');
  await expect(page.getByText('请输入 7 到 3650 之间的数字')).toBeVisible();
  await audit.fill('3651');
  await expect(page.getByText('请输入 7 到 3650 之间的数字')).toBeVisible();
  await audit.fill('180');
  await expect(page.getByText('请输入 7 到 3650 之间的数字')).toHaveCount(0);
  await checkPageHealth(page);
});

test('filters the operation log by category and pages it ten at a time', async ({ page }, testInfo) => {
  await login(page);
  const prefix = uniqueName('audit', testInfo);
  for (let i = 0; i < 12; i++) await createFolderApi(page, `${prefix}-${i}`);
  await page.goto('/status');
  await expect(page.getByText(/保留 \d+ 天/)).toBeVisible();

  await page.getByRole('button', { name: '操作类型' }).click();
  await page.getByRole('menuitemradio', { name: '文件夹', exact: true }).click();

  const rows = page.locator('.list-row.audit-row:not(.list-head)');
  await expect(rows).toHaveCount(10);
  const more = page.getByRole('button', { name: '加载更多', exact: true });
  await expect(more).toBeVisible();
  await more.click();
  await expect.poll(async () => rows.count()).toBeGreaterThan(10);
  for (const row of await rows.all()) {
    await expect(row.locator('.c-action')).toHaveText(/^(创建|重命名|移动|删除)文件夹$/);
  }
  await checkPageHealth(page);
});
