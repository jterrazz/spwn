import { test, expect } from "../fixtures/app";

test.describe("Agents page", () => {
  test.beforeEach(async ({ page, api }) => {
    // Ensure examples are installed (agents exist)
    await api.installExample("matrix");
    await api.installExample("startup");
    await page.goto("/");
    await page.waitForTimeout(2000);
  });

  test("shows agents list", async ({ page, app }) => {
    await app.goToAgents();
    await page.waitForTimeout(1000);

    // Should list agents installed from examples
    await expect(page.getByText("Neo")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("ceo")).toBeVisible();
  });

  test("clicking an agent shows their profile", async ({ page, app }) => {
    await app.goToAgents();
    await page.waitForTimeout(1000);

    // Click on Neo
    await page.getByText("Neo").first().click();
    await page.waitForTimeout(1000);

    // Should show the agent detail page with persona info
    await expect(page.getByRole("heading", { name: "Neo" })).toBeVisible({ timeout: 5000 });
  });

  test("agent profile shows the mind layers", async ({ page, app }) => {
    await app.goToAgents();
    await page.waitForTimeout(1000);
    await page.getByText("Neo").first().click();
    await page.waitForTimeout(1000);

    // Should show the mind layer directories (core, skills, knowledge, etc.)
    // The exact UI varies but should display the profile structure
    await expect(page.getByText(/core|persona|identity/i)).toBeVisible({ timeout: 5000 });
  });
});
