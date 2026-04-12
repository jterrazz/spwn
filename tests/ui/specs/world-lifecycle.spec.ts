import { test, expect } from "../fixtures/app";

test.describe("World lifecycle (requires Docker)", () => {
  test.beforeEach(async ({ api }) => {
    await api.destroyAll();
    await api.installExample("matrix");
  });

  test("spawn → appears in UI → destroy → disappears", async ({ page, api }) => {
    // Spawn via API
    const result = await api.spawnWorld("matrix", "Neo");
    const worldId = result.Universe.id;
    expect(worldId).toMatch(/^spwn-world-/);

    // Verify in UI
    await page.goto("/");
    await page.waitForTimeout(3000);
    await expect(page.getByText("Neo")).toBeVisible({ timeout: 10_000 });

    // Destroy
    await api.destroyWorld(worldId);
    await page.waitForTimeout(6000);

    // Verify gone
    const worlds = await api.worlds();
    expect(worlds.find((w) => w.id === worldId)).toBeUndefined();
  });

  test("multi-agent world shows all agents", async ({ page, api }) => {
    await api.installExample("startup");
    await api.spawnWorld("startup", undefined, [
      { name: "ceo", role: "chief" },
      { name: "devops", role: "worker" },
      { name: "analyst", role: "worker" },
    ]);

    await page.goto("/");
    await page.waitForTimeout(3000);

    // Select the world
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(1500);

    // Detail panel should list all agents
    await expect(page.getByText("ceo")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("devops")).toBeVisible();
    await expect(page.getByText("analyst")).toBeVisible();
  });

  test("world detail page loads correctly", async ({ page, api }) => {
    const result = await api.spawnWorld("matrix", "Neo");
    const worldId = result.Universe.id;

    // Navigate directly to the world detail
    await page.goto(`/world/${worldId}`);
    await page.waitForTimeout(3000);

    // Should show the world name and agent info
    await expect(page.getByText("Neo")).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText(/running|idle/i)).toBeVisible();
  });
});
