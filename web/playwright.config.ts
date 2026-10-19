import { defineConfig } from "@playwright/test"

export default defineConfig({
  testDir: "./tests",
  workers: 1,
  retries: 0,
  timeout: 45_000,
  reporter: "list",
  use: {
    browserName: "chromium",
    headless: true,
    viewport: { width: 1440, height: 1000 },
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
  },
})
