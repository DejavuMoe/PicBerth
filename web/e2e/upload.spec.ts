import {
  checkPageHealth, expect, finishUploads, libraryItem, login, samplePng, storedImages, test, uploadRow,
} from './harness/test-base.js';

test('cancels an in-flight upload and retries a transient 503', async ({ page }, testInfo) => {
  await login(page);
  const cancelFile = {
    name: `cancel-${testInfo.project.name}-${Date.now()}.png`,
    mimeType: 'image/png', buffer: samplePng(),
  };
  const started = Promise.withResolvers<void>();
  const held = Promise.withResolvers<void>();
  await page.route('**/api/v1/images', async (route) => {
    if (route.request().postDataBuffer()?.includes(Buffer.from(cancelFile.name))) {
      started.resolve();
      await held.promise;
      // Settle Playwright's interception only after the browser reports the product's abort.
      if (route.request().failure()) {
        await route.abort();
        return;
      }
    }
    await route.continue();
  });

  try {
    await page.locator('input[type="file"]').setInputFiles(cancelFile);
    await started.promise;
    const aborted = page.waitForEvent('requestfailed', {
      predicate: (request) => request.postDataBuffer()?.includes(Buffer.from(cancelFile.name)) ?? false,
      timeout: 5_000,
    });
    await page.getByRole('button', { name: `取消 ${cancelFile.name}`, exact: true }).click();
    // The product must abort the XHR; releasing the route must not abort it for the product.
    expect((await aborted).failure()?.errorText).toContain('ERR_ABORTED');
    held.resolve();
    await expect(uploadRow(page, cancelFile.name)).toHaveCount(0);
  } finally {
    held.resolve();
    await page.unrouteAll({ behavior: 'wait' });
  }
  expect(await storedImages(page, cancelFile.name)).toEqual([]);
  await expect(libraryItem(page, cancelFile.name)).toHaveCount(0);

  const retryFile = {
    name: `retry-${testInfo.project.name}-${Date.now()}.png`,
    mimeType: 'image/png', buffer: samplePng(),
  };
  let attempts = 0;
  await page.route('**/api/v1/images', async (route) => {
    if (route.request().postDataBuffer()?.includes(Buffer.from(retryFile.name)) && ++attempts === 1) {
      await route.fulfill({ status: 503, contentType: 'application/json',
        body: JSON.stringify({ code: 'service-unavailable', message: 'Transient backend outage' }) });
    } else {
      await route.continue();
    }
  });
  await page.locator('input[type="file"]').setInputFiles(retryFile);
  const retryButton = uploadRow(page, retryFile.name).getByRole('button', { name: '重试', exact: true });
  await expect(retryButton).toBeVisible();
  const uploaded = page.waitForResponse((response) => response.url().endsWith('/api/v1/images')
    && response.request().method() === 'POST' && response.status() === 201);
  await retryButton.click();
  const image = await (await uploaded).json();
  expect(attempts).toBe(2);
  expect(await storedImages(page, retryFile.name)).toMatchObject([
    { id: image.id, filename: retryFile.name, state: 'active', duplicate: false },
  ]);
  await finishUploads(page, [retryFile.name]);
  await expect(libraryItem(page, retryFile.name)).toBeVisible();
  await page.unrouteAll({ behavior: 'wait' });
  await page.reload();
  await expect(libraryItem(page, retryFile.name)).toBeVisible();
  await checkPageHealth(page);
});
