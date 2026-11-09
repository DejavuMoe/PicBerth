import { checkPageHealth, expect, login, test } from './harness/test-base.js';

test('protects settings drafts, discards on confirmation, and persists saves', async ({ page }, testInfo) => {
  await login(page);
  // Desktop and tablet use the side navigation; phones use the five-tab bar.
  const navigation = testInfo.project.name === 'mobile' ? 'nav.tabbar' : 'nav.side';
  const settingsLink = page.locator(`${navigation} a[href="/settings"]`);
  const libraryLink = page.locator(`${navigation} a[href="/library"]`);
  async function goToSettings() {
    await settingsLink.click();
    await expect(page).toHaveURL(/\/settings$/);
  }
  async function storedRetention() {
    const response = await page.request.get('/api/v1/settings');
    expect(response.ok()).toBe(true);
    return (await response.json()).trashRetentionDays;
  }
  const input = page.locator('#s-trash');
  const dirty = page.getByRole('region', { name: '未保存的更改' });
  const guard = page.getByRole('dialog', { name: '离开设置？' });
  await goToSettings();
  await expect(input).toBeVisible();
  const baseline = await input.inputValue();
  const draft = baseline === '45' ? '46' : '45';

  // 留下 keeps the draft and stays on the page.
  await input.fill(draft);
  await expect(dirty.getByText('有未保存的更改')).toBeVisible();
  await libraryLink.click();
  await expect(guard).toBeVisible();
  await guard.getByRole('button', { name: '留下', exact: true }).click();
  await expect(guard).toHaveCount(0);
  await expect(page).toHaveURL(/\/settings$/);
  await expect(input).toHaveValue(draft);
  expect(await storedRetention()).toBe(Number(baseline));

  // 放弃 in the save bar resets the form; leaving is then unguarded.
  await dirty.getByRole('button', { name: '放弃', exact: true }).click();
  await expect(dirty).toHaveCount(0);
  await expect(input).toHaveValue(baseline);
  await libraryLink.click();
  await expect(page).toHaveURL(/\/library$/);
  await expect(guard).toHaveCount(0);

  // 放弃更改 in the guard leaves without saving.
  await goToSettings();
  await input.fill(draft);
  await libraryLink.click();
  await guard.getByRole('button', { name: '放弃更改', exact: true }).click();
  await expect(page).toHaveURL(/\/library$/);
  expect(await storedRetention()).toBe(Number(baseline));

  // 保存 persists across navigation and reload.
  await goToSettings();
  await expect(input).toHaveValue(baseline);
  await input.fill(draft);
  await dirty.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByText('设置已保存')).toBeVisible();
  await expect(dirty).toHaveCount(0);
  expect(await storedRetention()).toBe(Number(draft));
  await libraryLink.click();
  await expect(page).toHaveURL(/\/library$/);
  await expect(guard).toHaveCount(0);
  await goToSettings();
  await page.reload();
  await expect(input).toHaveValue(draft);

  await input.fill(baseline);
  await dirty.getByRole('button', { name: '保存', exact: true }).click();
  await expect(dirty).toHaveCount(0);
  expect(await storedRetention()).toBe(Number(baseline));
  await checkPageHealth(page);
});
