import { defineConfig } from '@playwright/test';
import os from 'node:os';
import path from 'node:path';

const port = process.env.PICBERTH_PORT || '5210';
const baseURL = `http://127.0.0.1:${port}`;

export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  failOnFlakyTests: !!process.env.CI,
  reporter: [['list']],
  outputDir: process.env.E2E_OUTPUT_DIR || path.join(os.tmpdir(), 'picberth-e2e-artifacts'),
  use: {
    baseURL,
    browserName: 'chromium',
    headless: true,
    reducedMotion: 'reduce',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'desktop',
      use: {
        viewport: { width: 1440, height: 900 },
      },
    },
    {
      name: 'mobile',
      use: {
        viewport: { width: 390, height: 844 },
        isMobile: true,
        hasTouch: true,
      },
    },
  ],
  webServer: {
    command: 'node ./e2e/harness/server.mjs',
    url: `${baseURL}/readyz`,
    reuseExistingServer: false,
    gracefulShutdown: { signal: 'SIGTERM', timeout: 15_000 },
    timeout: 30_000,
    env: {
      PICBERTH_PORT: port,
    },
  },
});
