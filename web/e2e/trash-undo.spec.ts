import {
  checkPageHealth, expect, finishUploads, libraryItem, login, samplePng, storedImages, test,
} from './harness/test-base.js';

test('persists batch trash and restores the same images through undo', async ({ page }, testInfo) => {
  const mobile = testInfo.project.name === 'mobile';
  await login(page);
  const files = ['a', 'b'].map((suffix) => ({
    name: `batch-${suffix}-${testInfo.project.name}-${Date.now()}.png`,
    mimeType: 'image/png', buffer: samplePng(),
  }));
  await page.locator('input[type="file"]').setInputFiles(files);
  await finishUploads(page, files.map((f) => f.name));

  const originals = [];
  if (mobile) {
    // Phones start selection mode from the header ⋯ menu; a tap then toggles a row.
    await page.getByRole('button', { name: '更多', exact: true }).click();
    await page.getByRole('menuitem', { name: '选择', exact: true }).click();
  }
  for (const file of files) {
    const stored = await storedImages(page, file.name);
    expect(stored).toHaveLength(1);
    originals.push(stored[0]!);
    const item = libraryItem(page, file.name);
    await expect(item).toBeVisible();
    if (mobile) {
      await item.click();
    } else {
      await item.hover();
      await page.getByRole('checkbox', { name: `选择 ${file.name}`, exact: true }).click();
    }
  }
  await expect(page.getByText('已选 2 张')).toBeVisible();
  const bulk = mobile ? page.getByRole('toolbar', { name: '批量操作' }) : page.locator('.head .bulk');
  await bulk.getByRole('button', { name: '移到回收站' }).click();
  const undoButton = page.locator('.toast').getByRole('button', { name: '撤销', exact: true });
  await expect(undoButton).toBeVisible();
  for (const image of originals) {
    await expect(libraryItem(page, image.filename)).toHaveCount(0);
    expect(await storedImages(page, image.filename, 'trashed')).toMatchObject([{ id: image.id, state: 'trashed' }]);
    expect(await storedImages(page, image.filename)).toEqual([]);
  }

  await undoButton.click();
  for (const image of originals) await expect(libraryItem(page, image.filename)).toBeVisible();
  await page.reload();
  for (const image of originals) {
    await expect(libraryItem(page, image.filename)).toBeVisible();
    expect(await storedImages(page, image.filename)).toMatchObject([{ id: image.id, state: 'active' }]);
    expect(await storedImages(page, image.filename, 'trashed')).toEqual([]);
  }
  await checkPageHealth(page);
});
