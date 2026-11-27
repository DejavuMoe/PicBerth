import { checkPageHealth, expect, login, samplePng, test, uniqueName } from './harness/test-base.js';

test('lands on 最近 and lists a fresh upload there with one-press copy', async ({ page, context }, testInfo) => {
  const mobile = testInfo.project.name === 'mobile';
  await context.grantPermissions(['clipboard-read', 'clipboard-write']).catch(() => {});
  await login(page, 'recent');
  await expect(page.getByRole('heading', { level: 1, name: '最近' })).toBeVisible();
  await expect(page).toHaveTitle('最近 - PicBerth');

  const name = uniqueName('recent', testInfo) + '.png';
  await page.locator('input[type="file"]').setInputFiles({ name, mimeType: 'image/png', buffer: samplePng() });

  // The queue is listed in place on 最近; the floating dock stays away.
  const queue = page.locator('.rc-queue');
  await expect(queue.locator('.up-row').filter({ hasText: name }).getByRole('button', { name: '查看', exact: true })).toBeVisible();
  await expect(page.locator('.dock')).toHaveCount(0);

  const today = page.getByRole('region', { name: '今天' });
  const row = today.locator('.rc-row').filter({ hasText: name });
  await expect(row).toBeVisible();

  await page.getByRole('group', { name: '复制为' }).getByRole('button', { name: 'Markdown', exact: true }).click();
  await row.getByRole('button', { name: `复制 ${name} 的链接` }).click();
  await expect(page.getByText('已复制 Markdown')).toBeVisible();

  await row.locator('.rc-name').click();
  const detail = page.locator('.detail');
  await expect(detail).toBeVisible();
  await expect(detail.getByText(name).first()).toBeVisible();
  if (!mobile) await page.keyboard.press('Escape');
  await checkPageHealth(page);
});

test('redirects the root to 最近 and groups 状态, 令牌 and 设置 under 系统', async ({ page }, testInfo) => {
  const navigation = testInfo.project.name === 'mobile' ? 'nav.tabbar' : 'nav.side';
  await login(page, 'recent');
  await page.goto('/');
  await expect(page).toHaveURL(/\/recent$/);

  await expect(page.locator(`${navigation} a[href="/tokens"]`)).toHaveCount(0);
  await page.locator(`${navigation} a[href="/status"]`).click();
  await expect(page).toHaveURL(/\/status$/);
  const tabs = page.locator('nav.sys-tabs');
  await expect(tabs.getByRole('link', { name: '状态' })).toHaveAttribute('aria-current', 'page');
  await tabs.getByRole('link', { name: '令牌' }).click();
  await expect(page).toHaveURL(/\/tokens$/);
  await expect(page.locator(`${navigation} a[href="/status"]`)).toHaveAttribute('aria-current', 'page');
  await checkPageHealth(page);
});
