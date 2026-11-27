import { createFolderApi, expect, login, samplePng, test, uniqueName } from './harness/test-base.js';

test('opens a palette result that lies beyond the first page of its folder', async ({ page }, testInfo) => {
  await login(page);
  const folder = await createFolderApi(page, uniqueName('deep', testInfo));
  const names = [1, 2, 3, 4, 5].map((n) => `${uniqueName('paged', testInfo)}-${n}.png`);
  for (const name of names) {
    const response = await page.request.post('/api/v1/images', {
      headers: { 'X-PicBerth-Client': 'web' },
      multipart: { folder: folder.id, file: { name, mimeType: 'image/png', buffer: samplePng() } },
    });
    expect(response.status(), await response.text()).toBe(201);
  }
  // Two images a page: the oldest upload is on the third page of its folder.
  await page.route('**/api/v1/images?*', async (route) => {
    const url = new URL(route.request().url());
    url.searchParams.set('limit', '2');
    await route.continue({ url: url.toString() });
  });

  await page.goto('/library');
  await expect(page.locator('#main')).toBeVisible();
  await page.keyboard.press('Control+k');
  await page.getByRole('combobox', { name: '搜索图片、文件夹或操作' }).fill(names[0]!);
  await expect(page.locator('#cmdk-list [role="option"]').filter({ hasText: names[0]! })).toBeVisible();
  await page.keyboard.press('Enter');

  await expect(page).toHaveURL(new RegExp(`/library/${folder.name}$`));
  await expect(page.locator('.detail')).toContainText(names[0]!);
  await page.unrouteAll({ behavior: 'wait' });
});
