import { test, expect } from "../fixtures/app";

test.describe("Worlds page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await page.waitForTimeout(2000);
  });

  test("shows the Worlds heading", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Worlds" })).toBeVisible();
  });

  test("sidebar shows navigation items", async ({ page }) => {
    await expect(page.getByRole("button", { name: "Architect" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Settings" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Worlds" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Agents" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Tools" })).toBeVisible();
  });

  test("shows example gallery when no worlds are running", async ({ page, api }) => {
    // Ensure no worlds running
    await api.destroyAll();
    await page.goto("/");
    await page.waitForTimeout(3000);

    // Should show the gallery with templates
    await expect(page.getByText("Pick a template")).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText("Startup")).toBeVisible();
    await expect(page.getByText("The Matrix")).toBeVisible();
  });

  test("shows planets when worlds exist", async ({ page, api }) => {
    // Spawn a world via API
    await api.installExample("matrix");
    await api.spawnWorld("matrix", "Neo");
    await page.goto("/");
    await page.waitForTimeout(3000);

    // Should show at least one planet with the "+" new world button
    await expect(page.getByText("New World")).toBeVisible({ timeout: 10_000 });
  });

  test("selecting a planet shows the detail panel", async ({ page, api }) => {
    await api.installExample("matrix");
    const result = await api.spawnWorld("matrix", "Neo");
    await page.goto("/");
    await page.waitForTimeout(3000);

    // Click the planet — the sidebar should show world details
    // Arrow keys navigate, or click the planet label
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(1500);

    // Detail panel should show agent name
    await expect(page.getByText("Neo")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(/Enter World/)).toBeVisible();
  });

  test("destroying a world removes it from the list", async ({ page, api }) => {
    await api.installExample("matrix");
    const result = await api.spawnWorld("matrix", "Neo");
    const worldId = result.Universe.id;

    await page.goto("/");
    await page.waitForTimeout(3000);

    // Destroy via API
    await api.destroyWorld(worldId);
    await page.waitForTimeout(6000); // wait for poll

    // Should show empty state or gallery
    await expect(page.getByText(/Pick a template|Give your agents/)).toBeVisible({ timeout: 10_000 });
  });
});
