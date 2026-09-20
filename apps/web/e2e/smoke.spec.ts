import { test, expect } from '@playwright/test';

test('landing reveals all chapters and CTA', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 2, name: /Your files become noise/ })).toBeVisible();
  await page.mouse.wheel(0, 20000);
  await page.waitForTimeout(1500);
  await expect(page.getByRole('link', { name: 'Read the guide' })).toBeVisible();
  for (const id of ['upload', 'obfuscate', 'shard', 'store', 'keyfile', 'integrity', 'drive', 'cta']) await expect(page.locator(`#${id}`)).toHaveCount(1);
});

test('reduced motion still shows headlines immediately', async ({ browser }) => {
  const ctx = await browser.newContext({ reducedMotion: 'reduce' });
  const page = await ctx.newPage();
  await page.goto('/');
  await expect(page.locator('#store h2')).toHaveText(/S3,\s+under\s+your\s+KMS\s+key/);
});

test('guide TOC navigates', async ({ page }) => {
  await page.goto('/guide');
  await page.getByRole('link', { name: /key file/i }).first().click();
  await expect(page).toHaveURL(/#/);
  await expect(page.locator('section#keyfile h2')).toBeVisible();
});

test('login page renders split layout', async ({ page }) => {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible();
  await expect(page.getByLabel('Email')).toBeVisible();
});

test('tour shows once on /files', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('auth_token', 'x'));
  await page.goto('/files');
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await page.reload();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await page.locator('#tour-help').click();
  await expect(page.getByRole('dialog')).toBeVisible();
});

test.describe('mobile nav', () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test('tour targets are unique on mobile', async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem('auth_token', 'x'));
    await page.goto('/files');
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.keyboard.press('Escape'); // dismiss the auto-opening tour overlay first
    await expect(page.getByRole('dialog')).toHaveCount(0);
    await page.getByRole('button', { name: 'Open menu' }).click();
    await expect(page.locator('#tour-help')).toHaveCount(1);
  });
});
