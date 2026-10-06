import { defineConfig, devices } from "@playwright/test";

const port = process.env.PLAYWRIGHT_PORT ?? "4173";
const baseURL = `http://127.0.0.1:${port}`;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  reporter: "line",
  use: {
    baseURL,
    trace: "on-first-retry",
    ...devices["iPhone 13"],
    browserName: "chromium",
  },
  webServer: {
    command: `bun run generate:api && bunx --bun vite --host 127.0.0.1 --port ${port} --strictPort`,
    url: baseURL,
    // Reusing an old Vite process can serve cached modules from before the current edits.
    reuseExistingServer: false,
  },
});
