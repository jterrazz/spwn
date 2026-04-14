import { test, expect } from "../fixtures/app";

test.describe("Example gallery", () => {
  test.beforeEach(async ({ page, api }) => {
    await api.destroyAll();
    await page.goto("/");
    await page.waitForTimeout(3000);
  });

  test("shows bundled examples when no worlds running", async ({ page }) => {
    // Gallery shows example cards with h3 headings
    await expect(page.getByRole("heading", { name: "Startup", level: 3 })).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole("heading", { name: "The Matrix", level: 3 })).toBeVisible();
  });

  test("shows Install & spawn buttons", async ({ page }) => {
    const buttons = page.getByRole("button", { name: "Install & spawn" });
    await expect(buttons.first()).toBeVisible({ timeout: 10_000 });
    // Should have one per example
    await expect(buttons).toHaveCount(5, { timeout: 5_000 });
  });

  test("shows agent badges on cards", async ({ page }) => {
    // Startup card should show ceo, devops, analyst
    await expect(page.getByText("ceo").first()).toBeVisible({ timeout: 10_000 });
  });

  test("shows CLI command preview", async ({ page }) => {
    await expect(page.getByText(/\$ spwn up startup/)).toBeVisible({ timeout: 10_000 });
  });

  test("install & spawn creates a world", async ({ page, api }) => {
    const buttons = page.getByRole("button", { name: /Install & spawn/ });
    await buttons.first().click();

    // Wait for the spawn to complete (builds image, starts container)
    await page.waitForTimeout(15_000);

    // Verify via API that a world was created
    const worlds = await api.worlds();
    expect(worlds.length).toBeGreaterThan(0);
  });
});
