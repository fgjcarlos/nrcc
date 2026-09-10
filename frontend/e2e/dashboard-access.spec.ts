import { expect, test } from '@playwright/test';
import { editableHostStatus, securityCenterConfig } from '../src/test/msw/fixtures';
import { envelope, login } from './helpers';

test('applies a mocked dashboard policy without displaying its secret', async ({ page }) => {
  let request = {};
  await login(page);
  await page.route('**/api/bootstrap/status', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope(editableHostStatus)) }));
  await page.route('**/api/config', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope(securityCenterConfig)) }));
  await page.route('**/api/settings/raw', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope({ writable: true, revision: { fingerprint: 'revision', algorithm: 'sha256' } })) }));
  await page.route('**/api/dashboards/discovery', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope({ packages: { state: 'available' }, flows: { state: 'available' }, legacy: { path: '/ui' }, flowFuse: [], uiBases: [] })) }));
  await page.route('**/api/dashboards/access', async (route) => { request = JSON.parse(route.request().postData() ?? '{}'); await route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope({ document: {} })) }); });
  await page.goto('/configuration'); await page.getByRole('button', { name: 'Authentication' }).click();
  await page.getByLabel('Access secret').fill('browser-secret'); await page.getByLabel('Username').last().fill('operator');
  await page.getByRole('button', { name: 'Apply dashboard access' }).click();
  await expect.poll(() => request).toMatchObject({ target: 'legacy', username: 'operator', expectedRevision: 'revision' });
  await expect(page.getByRole('complementary', { name: 'Redacted dashboard policy preview' })).not.toContainText('browser-secret');
});
