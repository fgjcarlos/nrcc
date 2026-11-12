import process from 'node:process'
import { defineConfig, devices } from '@playwright/test'

const BASE_URL = 'http://127.0.0.1:4173'
const A11Y_SPECS = /a11y.*\.spec\.ts$/

function themeStorageState(theme: 'dark' | 'light') {
  return {
    cookies: [],
    origins: [{ origin: BASE_URL, localStorage: [{ name: 'cc-theme', value: theme }] }],
  }
}

export default defineConfig({
  testDir: './e2e',
  testIgnore: '**/stack.spec.ts',
  timeout: 30_000,
  expect: { timeout: 10_000 },
  fullyParallel: true,
  workers: 1,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI
    ? [['github'], ['html', { open: 'never', outputFolder: 'playwright-report' }]]
    : [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  use: {
    baseURL: BASE_URL,
    trace: 'retain-on-failure',
  },
  webServer: {
    command: 'npm run dev -- --host 127.0.0.1 --port 4173',
    url: BASE_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
  // Issue #889: the theme is pinned per project through the same
  // localStorage key main.tsx/useTheme read before the first render, and
  // the emulated colour scheme matches it. The dark desktop project runs
  // the whole fixture suite; light desktop and the mobile viewport run
  // only the accessibility specs so the suite does not triple in length.
  projects: [
    {
      name: 'chromium-dark',
      metadata: { theme: 'corporateDark' },
      use: {
        ...devices['Desktop Chrome'],
        colorScheme: 'dark',
        storageState: themeStorageState('dark'),
      },
    },
    {
      name: 'chromium-light',
      testMatch: A11Y_SPECS,
      metadata: { theme: 'corporateLight' },
      use: {
        ...devices['Desktop Chrome'],
        colorScheme: 'light',
        storageState: themeStorageState('light'),
      },
    },
    {
      name: 'mobile-light',
      testMatch: A11Y_SPECS,
      metadata: { theme: 'corporateLight', mobile: true },
      use: {
        // Pixel 7: 412px wide, Chromium, touch + mobile viewport.
        ...devices['Pixel 7'],
        colorScheme: 'light',
        storageState: themeStorageState('light'),
      },
    },
  ],
})
