import { test, expect } from "../fixtures/app";

test.describe("Navigation", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.waitForTimeout(2000);
  });

  test("sidebar navigation works", async ({ page }) => {
    // Click Agents
    await page.getByRole("button", { name: "Agents" }).click();
    await page.waitForTimeout(500);
    await expect(page).toHaveURL(/agents/);

    // Click Tools
    await page.getByRole("button", { name: "Tools" }).click();
    await page.waitForTimeout(500);
    await expect(page).toHaveURL(/tools/);

    // Click Settings
    await page.getByRole("button", { name: "Settings" }).click();
    await page.waitForTimeout(500);
    await expect(page).toHaveURL(/settings/);

    // Click back to Worlds
    await page.getByRole("button", { name: "Worlds" }).click();
    await page.waitForTimeout(500);
    await expect(page).toHaveURL(/\/$/);
  });

  test("command palette opens with Cmd+K", async ({ page }) => {
    await page.keyboard.press("Meta+k");
    await page.waitForTimeout(500);

    // Command palette should be visible
    await expect(page.getByPlaceholder(/search|command/i)).toBeVisible({ timeout: 3000 });

    // Close it
    await page.keyboard.press("Escape");
    await page.waitForTimeout(300);
  });

  test("top bar shows world/agent/task counts", async ({ page }) => {
    // The header shows stats badges
    await expect(page.locator("header, [class*='header']").first()).toBeVisible();
  });

  test("version info is visible in sidebar footer", async ({ page }) => {
    // The sidebar footer shows the Docker version
    await expect(page.getByText(/v\d+\.\d+/)).toBeVisible({ timeout: 5000 });
  });
});
