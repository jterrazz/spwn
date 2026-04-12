import { test, expect } from "../fixtures/app";

test.describe("Agents page", () => {
  test.beforeEach(async ({ api }) => {
    await api.installExample("matrix");
    await api.installExample("startup");
  });

  test("shows agents list", async ({ page }) => {
    await page.goto("/");
    await page.waitForTimeout(2000);
    await page.getByRole("button", { name: "Agents", exact: true }).first().click();
    await page.waitForTimeout(1500);

    await expect(page.getByText("Neo")).toBeVisible({ timeout: 5000 });
  });

  test("clicking an agent navigates to their detail", async ({ page }) => {
    await page.goto("/");
    await page.waitForTimeout(2000);
    await page.getByRole("button", { name: "Agents", exact: true }).first().click();
    await page.waitForTimeout(1500);

    await page.getByText("Neo").first().click();
    await page.waitForTimeout(2000);

    await expect(page).toHaveURL(/agents/, { timeout: 5000 });
  });

  test("agent detail page loads directly", async ({ page }) => {
    await page.goto("/agents/Neo");
    await page.waitForTimeout(2000);

    await expect(page.getByText("Neo").first()).toBeVisible({ timeout: 5000 });
  });
});
