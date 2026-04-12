import { test, expect } from "../fixtures/app";

test.describe("Example gallery", () => {
  test.beforeEach(async ({ page, api }) => {
    // Ensure clean state — no worlds running
    await api.destroyAll();
    await page.goto("/");
    await page.waitForTimeout(3000);
  });

  test("shows all 5 bundled examples", async ({ page }) => {
    await expect(page.getByText("Startup")).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText("The Matrix")).toBeVisible();
    await expect(page.getByText("Paperclip Factory")).toBeVisible();
    await expect(page.getByText("Research Lab")).toBeVisible();
    await expect(page.getByText("Macrohard")).toBeVisible();
  });

  test("startup is the first example (featured)", async ({ page }) => {
    // Startup should appear before Matrix in the DOM
    const cards = page.locator('[class*="rounded-2xl"]').filter({ hasText: /Install & spawn/ });
    const firstCard = cards.first();
    await expect(firstCard).toContainText("Startup", { timeout: 10_000 });
  });

  test("each example shows agent badges", async ({ page }) => {
    // Startup should list ceo, devops, analyst
    const startupCard = page.locator("text=Startup").locator("..").locator("..");
    await expect(startupCard.getByText("ceo")).toBeVisible({ timeout: 10_000 });
  });

  test("each example shows a CLI command preview", async ({ page }) => {
    // Cards should show the command that will run
    await expect(page.getByText(/spwn up -c startup/)).toBeVisible({ timeout: 10_000 });
  });

  test("install & spawn creates a world and navigates to it", async ({ page, api }) => {
    // Click the first "Install & spawn" button (should be startup)
    const installButtons = page.getByRole("button", { name: /Install & spawn/ });
    await installButtons.first().click();

    // Should navigate to the world detail page or show the world
    await expect(page.getByText(/running|idle|Starting/)).toBeVisible({ timeout: 30_000 });

    // Verify the world was actually created
    const worlds = await api.worlds();
    expect(worlds.length).toBeGreaterThan(0);
  });
});
