import { test, expect } from "../fixtures/app";

test.describe("API health", () => {
  test("API is connected (not showing offline banner)", async ({ page }) => {
    await page.goto("/");
    await page.waitForTimeout(3000);

    // Should NOT show the "Waiting for Docker" or "API OFFLINE" screen
    await expect(page.getByText("Waiting for Docker")).not.toBeVisible({ timeout: 5000 });
    await expect(page.getByText("API OFFLINE")).not.toBeVisible();
  });

  test("API version endpoint returns data", async ({ api }) => {
    const version = await api.get<{ current: string }>("/api/version");
    expect(version.current).toBeTruthy();
  });

  test("API examples endpoint returns 5 examples", async ({ api }) => {
    const data = await api.get<{ examples: Array<{ slug: string }> }>("/api/examples");
    expect(data.examples).toHaveLength(5);
    expect(data.examples[0].slug).toBe("startup");
  });

  test("API agents endpoint returns installed agents", async ({ api }) => {
    await api.installExample("matrix");
    const agents = await api.get<Array<{ name: string }>>("/api/agents");
    const names = agents.map((a) => a.name);
    expect(names).toContain("Neo");
  });

  test("doctor check passes (Docker running)", async ({ api }) => {
    const output = api.cli("doctor");
    expect(output).toContain("Docker");
    expect(output).toContain("✓");
  });
});
