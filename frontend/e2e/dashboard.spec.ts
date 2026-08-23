import { expect, test } from '@playwright/test';
import { login } from './helpers';

const viewports = [
  { name: 'phone', width: 375, height: 812 },
  { name: 'tablet', width: 768, height: 1024 },
  { name: 'sm', width: 640, height: 900 },
  { name: 'md', width: 768, height: 900 },
  { name: 'lg', width: 1280, height: 900 },
];

test.describe('Dashboard redesign', () => {
  for (const viewport of viewports) {
    test(`keeps Runtime, CPU, and Memory ahead of Disk and Backups at ${viewport.name}`, async ({ page }) => {
      await page.setViewportSize(viewport);
      await login(page);
      const labels = await page.locator('[data-dashboard-status-card]').evaluateAll(cards => cards.map(card => card.getAttribute('data-dashboard-status-card')));
      expect(labels.slice(0, 3)).toEqual(['Runtime', 'CPU', 'Memory']);
      await expect(page.getByText('Disk Usage')).toBeVisible();
      await expect(page.getByText('Backups locales', { exact: true })).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    });
  }

  for (const theme of ['corporateDark', 'corporateLight']) {
    test(`does not clip dashboard content in ${theme}`, async ({ page }) => {
      await login(page);
      await page.evaluate((name) => document.documentElement.setAttribute('data-theme', name), theme);
      await expect(page.getByTestId('security-posture-card')).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    });
  }

  test('supports Runtime actions and keyboard traversal into posture chips', async ({ page }) => {
    await login(page);
    await page.getByRole('button', { name: 'Restart' }).press('Enter');
    await expect(page.getByRole('heading', { name: /Reiniciar Node-RED/i })).toBeVisible();
    await page.keyboard.press('Escape');
    const open = page.locator('[data-dashboard-status-card="Runtime"]').getByRole('button', { name: 'Open', exact: true });
    await open.focus();
    await expect(open).toBeFocused();
    await page.keyboard.press('Tab');
    await expect(page.getByRole('status', { name: /Encryption key:/i })).toBeFocused();
  });

  test('renders healthy, degraded, and critical posture states for admins', async ({ page }) => {
    await login(page, 'healthy');
    await expect(page.getByRole('status', { name: /Encryption key: configured, healthy/i })).toBeVisible();
    await login(page, 'degraded');
    await expect(page.getByRole('status', { name: /Sessions: 3 active, degraded/i })).toBeVisible();
    await login(page, 'critical');
    await expect(page.getByRole('status', { name: /Encryption key: not configured, critical/i })).toBeVisible();
  });
});
