import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  use: { baseURL: 'http://localhost:5183', headless: true },
  webServer: { command: 'npm run dev -- --port 5183', port: 5183, reuseExistingServer: true },
});
