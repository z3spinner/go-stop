const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './e2e',
  timeout: 30000,
  use: {
    baseURL: 'http://localhost:8080',
    headless: true,
    timezoneId: 'Europe/Paris',
  },
  reporter: [['list'], ['html', { open: 'never' }]],
  webServer: {
    // Builds the SvelteKit app into web/build, then runs the Go server that serves it.
    // Requires Postgres + VAPID env (e.g. `docker-compose up -d db migrations` first,
    // or an already-running stack — reuseExistingServer picks that up).
    command: 'npm run build && go run .',
    url: 'http://localhost:8080',
    reuseExistingServer: true,
    timeout: 180000,
  },
});
