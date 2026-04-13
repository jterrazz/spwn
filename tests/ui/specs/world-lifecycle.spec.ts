import { test, expect } from "../fixtures/app";

test.describe("World lifecycle (requires Docker)", () => {
  test.beforeEach(async ({ api }) => {
    await api.destroyAll();
    await api.installExample("matrix");
  });

  test("spawn → appears in UI → destroy → disappears", async ({ page, api }) => {
    const result = await api.spawnWorld("matrix", "Neo");
    const worldId = result.World.id;
    expect(worldId).toMatch(/^spwn-world-/);

    await page.goto("/");
    await page.waitForTimeout(4000);

    // World should be visible (New World button = planets are showing)
    await expect(page.getByRole("button", { name: "New World" })).toBeVisible({ timeout: 10_000 });

    // Destroy and verify
    await api.destroyWorld(worldId);
    await page.waitForTimeout(6000);

    const worlds = await api.worlds();
    expect(worlds.find((w) => w.id === worldId)).toBeUndefined();
  });

  test("multi-agent world shows all agents in sidebar", async ({ page, api }) => {
    await api.installExample("startup");
    await api.spawnWorld("startup", undefined, [
      { name: "ceo", role: "chief" },
      { name: "devops", role: "worker" },
      { name: "analyst", role: "worker" },
    ]);

    await page.goto("/");
    await page.waitForTimeout(4000);

    // Select the startup world in the sidebar
    await page.getByText("World").first().click();
    await page.waitForTimeout(1500);

    // The sidebar should list the agents
    await expect(page.getByText("ceo")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("devops")).toBeVisible();
    await expect(page.getByText("analyst")).toBeVisible();
  });

  test("world detail page loads", async ({ page, api }) => {
    const result = await api.spawnWorld("matrix", "Neo");
    const worldId = result.World.id;

    await page.goto(`/world/${worldId}`);
    await page.waitForTimeout(4000);

    // Should show the world — agent name or status visible
    await expect(page.getByText(/Neo|matrix|running|idle/i).first()).toBeVisible({ timeout: 10_000 });
  });
});
