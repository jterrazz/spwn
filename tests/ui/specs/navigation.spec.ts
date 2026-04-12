import { test, expect } from "../fixtures/app";

test.describe("Navigation", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.waitForTimeout(2000);
  });

  test("sidebar navigation changes page content", async ({ page }) => {
    // Click Agents — content should change
    await page.getByRole("button", { name: "Agents", exact: true }).first().click();
    await page.waitForTimeout(1000);
    await expect(page.getByRole("heading", { name: "Agents" })).toBeVisible({ timeout: 5000 });

    // Click Tools
    await page.getByRole("button", { name: "Tools", exact: true }).first().click();
    await page.waitForTimeout(1000);
    await expect(page.getByRole("heading", { name: "Tools" })).toBeVisible({ timeout: 5000 });

    // Click back to Worlds
    await page.getByRole("button", { name: "Worlds", exact: true }).first().click();
    await page.waitForTimeout(1000);
    await expect(page.getByRole("heading", { name: "Worlds", level: 1 })).toBeVisible({ timeout: 5000 });
  });

  test("command palette opens with Cmd+K", async ({ page }) => {
    await page.keyboard.press("Meta+k");
    await page.waitForTimeout(500);

    await expect(page.getByText(/Search for a command/i)).toBeVisible({ timeout: 3000 });

    await page.keyboard.press("Escape");
    await page.waitForTimeout(300);
  });

  test("header shows stats buttons", async ({ page }) => {
    await expect(page.locator('button:has-text("WORLDS")')).toBeVisible({ timeout: 5000 });
  });

  test("Docker version is visible in sidebar", async ({ page }) => {
    await expect(page.getByRole("button", { name: /Docker status/ })).toBeVisible({ timeout: 5000 });
  });
});
