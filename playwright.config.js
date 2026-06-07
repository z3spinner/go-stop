const { defineConfig } = require('@playwright/test');

// When E2E_BASE_URL is set (the containerized `gostop-e2e` stack, see
// docker-compose.e2e.yml) the app is already running and served by name on the
// compose network, so Playwright targets it directly and starts no webServer.
// Unset (a bare local `npx playwright test`) keeps the old behaviour: build the
// SvelteKit app and run the Go server on :8080.
const baseURL = process.env.E2E_BASE_URL || 'http://localhost:8080';

module.exports = defineConfig({
  testDir: './e2e',
  timeout: 30000,
  use: {
    baseURL,
    headless: true,
    timezoneId: 'Europe/Paris',
  },
  reporter: [['list'], ['html', { open: 'never' }]],
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : {
        // Builds the SvelteKit app into web/build, then runs the Go server that serves it.
        // Requires Postgres + VAPID env (e.g. `docker-compose up -d db migrations` first,
        // or an already-running stack — reuseExistingServer picks that up).
        command: 'npm run build && go run .',
        url: 'http://localhost:8080',
        reuseExistingServer: true,
        timeout: 180000,
      },
});
