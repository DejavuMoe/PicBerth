import type { Page } from '@playwright/test';
import {
  checkPageHealth, createFolderApi, expect, finishUploads, libraryItem, login, samplePng, storedFolders,
  storedImages, test, uniqueName, uploadRow, useListView,
} from './harness/test-base.js';

function tree(page: Page) {
  return page.locator('.side .tree-body, .popover');
}

async function openTree(page: Page, mobile: boolean) {
  if (!mobile) {
    await expect(page.locator('.side .tree-body')).toBeVisible();
    return;
  }
  if (await page.locator('.popover .tree-head').isVisible().catch(() => false)) return;
  await page.getByRole('button', { name: '文件夹', exact: true }).click();
  await expect(page.locator('.popover .tree-head')).toBeVisible();
}

async function createRoot(page: Page, mobile: boolean, name: string) {
  await openTree(page, mobile);
  await page.getByRole('button', { name: '新建文件夹', exact: true }).click();
  const input = page.getByLabel('文件夹名称');
  await input.fill(name);
  await input.press('Enter');
  await expect(tree(page).locator('.tn-main').filter({ hasText: name })).toBeVisible();
}

async function createChild(page: Page, mobile: boolean, parent: string, name: string) {
  await openTree(page, mobile);
  await page.getByRole('button', { name: `${parent} 的更多操作` }).click();
  await page.getByRole('menuitem', { name: '新建子文件夹' }).click();
  const input = page.getByLabel('文件夹名称');
  await input.fill(name);
  await input.press('Enter');
  await expect(tree(page).locator('.tn-main').filter({ hasText: name })).toBeVisible();
}

async function openFolder(page: Page, mobile: boolean, name: string) {
  await openTree(page, mobile);
  await tree(page).locator('.tn-main').filter({ hasText: name }).click();
}

async function openFolderMenu(page: Page, mobile: boolean, name: string) {
  await openTree(page, mobile);
  await page.getByRole('button', { name: `${name} 的更多操作` }).click();
}

test('creates, renames and validates folders inline, and shows the missing-folder state', async ({ page }, testInfo) => {
  const mobile = testInfo.project.name === 'mobile';
  await login(page);
  const base = uniqueName('base', testInfo);
  const child = uniqueName('child', testInfo);

  await createRoot(page, mobile, base);
  await createChild(page, mobile, base, child);
  await openFolder(page, mobile, child);
  await expect(page).toHaveURL(new RegExp(`/library/${base}/${child}$`));
  await expect(page.getByRole('heading', { level: 1, name: child })).toBeVisible();

  // Renaming an ancestor keeps the address on the folder being viewed.
  await openFolderMenu(page, mobile, base);
  await page.getByRole('menuitem', { name: '重命名' }).click();
  const input = page.getByLabel('文件夹名称');
  await expect(input).toHaveValue(base);
  const renamed = `${base}x`;
  await input.fill(renamed);
  await input.press('Enter');
  await expect(page).toHaveURL(new RegExp(`/library/${renamed}/${child}$`));
  await openTree(page, mobile);
  await expect(tree(page).locator('.tn-main').filter({ hasText: renamed })).toBeVisible();

  // A duplicate name is refused inline in the tree, not by a dialog.
  await openFolderMenu(page, mobile, renamed);
  await page.getByRole('menuitem', { name: '新建子文件夹' }).click();
  await page.getByLabel('文件夹名称').fill(child);
  await page.getByLabel('文件夹名称').press('Enter');
  await expect(page.locator('.tn-err')).toHaveText('已有同名文件夹');
  await page.getByLabel('文件夹名称').press('Escape');

  await page.goto('/library/no-such-folder-e2e');
  await expect(page.getByText('文件夹不存在')).toBeVisible();
  await page.getByRole('button', { name: '返回图片' }).click();
  await expect(page).toHaveURL(/\/library$/);
  await checkPageHealth(page);
});

test('enforces the depth limit and folder delete rules, and undoes a delete', async ({ page }, testInfo) => {
  const mobile = testInfo.project.name === 'mobile';
  await login(page);
  const a = await createFolderApi(page, uniqueName('a', testInfo));
  const b = await createFolderApi(page, uniqueName('b', testInfo), a.id);
  const c = await createFolderApi(page, uniqueName('c', testInfo), b.id);
  await page.goto(`/library/${a.name}/${b.name}/${c.name}`);
  await expect(page.getByRole('heading', { level: 1, name: c.name })).toBeVisible();

  // Depth 3: a fourth level cannot be created.
  await openFolderMenu(page, mobile, c.name);
  const newChild = page.getByRole('menuitem', { name: /新建子文件夹/ });
  await expect(newChild).toBeDisabled();
  await expect(newChild).toContainText('最多 3 层');
  await expect(page.getByRole('menuitem', { name: '删除' })).toBeEnabled();
  await page.keyboard.press('Escape');

  // A folder with a subfolder refuses deletion.
  await openFolderMenu(page, mobile, a.name);
  const del = page.getByRole('menuitem', { name: '删除' });
  await expect(del).toBeDisabled();
  await expect(del).toContainText('不为空');
  await page.keyboard.press('Escape');

  // An empty folder is deleted at once and can be brought back from the toast.
  await openFolderMenu(page, mobile, c.name);
  await page.getByRole('menuitem', { name: '删除' }).click();
  await expect(page).toHaveURL(new RegExp(`/library/${a.name}/${b.name}$`));
  const undo = page.locator('.toast').getByRole('button', { name: '撤销', exact: true });
  await expect(undo).toBeVisible();
  await expect(tree(page).locator('.tn-main').filter({ hasText: c.name })).toHaveCount(0);
  await undo.click();
  await openTree(page, mobile);
  await expect(tree(page).locator('.tn-main').filter({ hasText: c.name })).toBeVisible();
  await checkPageHealth(page);
});

test('moves folders only where the depth allows and keeps the address in step', async ({ page }, testInfo) => {
  const mobile = testInfo.project.name === 'mobile';
  await login(page);
  const a = await createFolderApi(page, uniqueName('a', testInfo));
  const b = await createFolderApi(page, uniqueName('b', testInfo), a.id);
  const c = await createFolderApi(page, uniqueName('c', testInfo), b.id);
  const m = await createFolderApi(page, uniqueName('m', testInfo));
  await page.goto(`/library/${a.name}/${b.name}/${c.name}`);
  await expect(page.getByRole('heading', { level: 1, name: c.name })).toBeVisible();

  // Moving A under M would make the deepest node level 4.
  await openFolderMenu(page, mobile, a.name);
  await page.getByRole('menuitem', { name: '移动到…' }).click();
  const blocked = page.getByRole('menuitemradio', { name: m.name });
  await expect(blocked).toBeDisabled();
  await expect(blocked).toContainText('最多 3 层');
  await page.keyboard.press('Escape');

  // B fits under M, and the open folder's address follows it.
  await openFolderMenu(page, mobile, b.name);
  await page.getByRole('menuitem', { name: '移动到…' }).click();
  await page.getByRole('menuitemradio', { name: m.name }).click();
  await expect(page.locator('.toast').getByText(`已将 ${b.name} 移到 图片 / ${m.name}`)).toBeVisible();
  await expect(page).toHaveURL(new RegExp(`/library/${m.name}/${b.name}/${c.name}$`));
  await expect(page.getByRole('heading', { level: 1, name: c.name })).toBeVisible();

  // Undo puts it back and the address returns with it.
  await page.locator('.toast').getByRole('button', { name: '撤销', exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/library/${a.name}/${b.name}/${c.name}$`));

  // The tree list agrees with the server.
  const names = (await storedFolders(page)).map((f) => f.name);
  expect(names).toContain(b.name);
  await checkPageHealth(page);
});

test('moves one image through the row menu and undoes it', async ({ page }, testInfo) => {
  const mobile = testInfo.project.name === 'mobile';
  test.skip(mobile, 'the row menu is hidden on phones');
  await useListView(page);
  await login(page);
  const target = await createFolderApi(page, uniqueName('moves', testInfo));
  const name = uniqueName('move', testInfo) + '.png';
  await page.locator('input[type="file"]').setInputFiles({ name, mimeType: 'image/png', buffer: samplePng() });
  await finishUploads(page, [name]);
  await expect(libraryItem(page, name)).toBeVisible();

  const item = libraryItem(page, name);
  await item.hover();
  await item.getByRole('button', { name: `${name} 的更多操作` }).click();
  await page.getByRole('menuitem', { name: '移动到…' }).click();
  await page.getByRole('menuitemradio', { name: target.name }).click();
  await expect(libraryItem(page, name)).toHaveCount(0);
  expect((await storedImages(page, name))[0]?.folderId).toBe(target.id);

  await page.locator('.toast').getByRole('button', { name: '撤销', exact: true }).click();
  await expect(libraryItem(page, name)).toBeVisible();
  expect((await storedImages(page, name))[0]?.folderId).toBeNull();
  await checkPageHealth(page);
});

test('drags an image onto a folder node', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name === 'mobile', 'the tree is a popover on phones');
  await login(page);
  const target = await createFolderApi(page, uniqueName('drag', testInfo));
  const name = uniqueName('dragged', testInfo) + '.png';
  await page.locator('input[type="file"]').setInputFiles({ name, mimeType: 'image/png', buffer: samplePng() });
  await finishUploads(page, [name]);
  await expect(libraryItem(page, name)).toBeVisible();
  await expect(page.locator('.side .tn-main').filter({ hasText: target.name })).toBeVisible();

  // A single drag carries the image id to the tree node through the product's own drag payload.
  await page.evaluate(({ filename, folder }) => {
    const row = [...document.querySelectorAll("table.table tbody tr, .jgrid .jt")].find((tr) => tr.textContent?.includes(filename));
    const node = [...document.querySelectorAll(".side .tn-main")].find((b) => b.textContent?.includes(folder));
    if (!row || !node) throw new Error("drag source or target missing");
    const dt = new DataTransfer();
    row.dispatchEvent(new DragEvent("dragstart", { bubbles: true, cancelable: true, dataTransfer: dt }));
    const tn = node.closest(".tn");
    if (!tn) throw new Error("tree row missing");
    tn.dispatchEvent(new DragEvent("dragover", { bubbles: true, cancelable: true, dataTransfer: dt }));
    tn.dispatchEvent(new DragEvent("drop", { bubbles: true, cancelable: true, dataTransfer: dt }));
  }, { filename: name, folder: target.name });

  await expect(libraryItem(page, name)).toHaveCount(0);
  expect((await storedImages(page, name))[0]?.folderId).toBe(target.id);
  await page.locator('.toast').getByRole('button', { name: '撤销', exact: true }).click();
  await expect(libraryItem(page, name)).toBeVisible();
  expect((await storedImages(page, name))[0]?.folderId).toBeNull();
  await checkPageHealth(page);
});

test('uploads into the current folder and finds it from the root with the folder column', async ({ page }, testInfo) => {
  const mobile = testInfo.project.name === 'mobile';
  await useListView(page);
  await login(page);
  const folder = await createFolderApi(page, uniqueName('up', testInfo));
  await page.goto(`/library/${folder.name}`);
  await expect(page.getByRole('heading', { level: 1, name: folder.name })).toBeVisible();

  const name = uniqueName('infolder', testInfo) + '.png';
  await page.locator('input[type="file"]').setInputFiles({ name, mimeType: 'image/png', buffer: samplePng() });
  await expect(uploadRow(page, name).getByText(`图片 / ${folder.name}`)).toBeVisible();
  await finishUploads(page, [name]);
  await expect(libraryItem(page, name)).toBeVisible();
  expect((await storedImages(page, name))[0]?.folderId).toBe(folder.id);

  // The root shows only its own images.
  await page.goto('/library');
  await expect(libraryItem(page, name)).toHaveCount(0);

  // Searching from the root covers the whole library and names the folder.
  if (mobile) {
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await page.getByLabel('搜索').fill(name);
  } else {
    await page.locator('.search input').fill(name);
  }
  const found = libraryItem(page, name);
  await expect(found).toBeVisible();
  if (mobile) {
    await expect(found).toContainText(folder.name);
  } else {
    await expect(page.getByRole('columnheader', { name: '文件夹' })).toBeVisible();
    await expect(found).toContainText(folder.name);
  }
  await checkPageHealth(page);
});
