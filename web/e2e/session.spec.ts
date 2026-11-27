import { expect, login, samplePng, storedImages, test, uniqueName, uploadRow } from './harness/test-base.js';

test('keeps queued uploads when the session ends and finishes them after signing in again', async ({ page }, testInfo) => {
  await login(page);
  const names = [1, 2, 3, 4, 5].map((n) => `${uniqueName('expired', testInfo)}-${n}.png`);

  let expired = true;
  let refused = 0;
  await page.route('**/api/v1/images', async (route) => {
    if (route.request().method() !== 'POST' || !expired) return route.continue();
    refused++;
    await route.fulfill({
      status: 401, contentType: 'application/problem+json',
      body: JSON.stringify({ type: 'urn:picberth:problem:unauthorized', title: 'Authentication required', status: 401, code: 'unauthorized' }),
    });
  });
  await page.locator('input[type="file"]').setInputFiles(names.map((name) => ({ name, mimeType: 'image/png', buffer: samplePng() })));

  await expect(page).toHaveURL(/\/login/);
  // Only the uploads already in flight met the expired session; the rest wait.
  expect(refused).toBe(3);
  expired = false;

  await page.getByLabel('用户名').fill('testadmin');
  await page.getByLabel('密码').fill('TestPass123!');
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await expect(page.locator('#main')).toBeVisible();

  for (let n = 0; n < 3; n++) {
    await page.locator('.upq .up-row, .dock .up-row').getByRole('button', { name: '重试', exact: true }).first().click();
  }
  for (const name of names) {
    await expect(uploadRow(page, name).getByRole('button', { name: '查看', exact: true })).toBeVisible();
    expect(await storedImages(page, name)).toHaveLength(1);
  }
  await page.unrouteAll({ behavior: 'wait' });
});

test('stays signed in when signing out fails', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name === 'mobile', 'the account menu is exercised on the desktop layout');
  await login(page);
  await page.route('**/api/v1/session', async (route) => {
    if (route.request().method() === 'DELETE') return route.abort();
    return route.continue();
  });
  await page.getByRole('button', { name: 'testadmin', exact: true }).click();
  await page.getByRole('menuitem', { name: '退出' }).click();
  await expect(page.getByText('退出失败，请重试')).toBeVisible();
  await expect(page).not.toHaveURL(/\/login/);
  await expect(page.locator('#main')).toBeVisible();
  await page.unrouteAll({ behavior: 'wait' });
});
