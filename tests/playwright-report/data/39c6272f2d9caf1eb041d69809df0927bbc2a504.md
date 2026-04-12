# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: api-health.spec.ts >> API health >> API agents endpoint returns installed agents
- Location: ui/specs/api-health.spec.ts:24:3

# Error details

```
Error: expect(received).toContain(expected) // indexOf

Expected value: "Neo"
Received array: ["analyst", "ceo", "devops", "neo"]
```

# Test source

```ts
  1  | import { test, expect } from "../fixtures/app";
  2  | 
  3  | test.describe("API health", () => {
  4  |   test("API is connected (not showing offline banner)", async ({ page }) => {
  5  |     await page.goto("/");
  6  |     await page.waitForTimeout(3000);
  7  | 
  8  |     // Should NOT show the "Waiting for Docker" or "API OFFLINE" screen
  9  |     await expect(page.getByText("Waiting for Docker")).not.toBeVisible({ timeout: 5000 });
  10 |     await expect(page.getByText("API OFFLINE")).not.toBeVisible();
  11 |   });
  12 | 
  13 |   test("API version endpoint returns data", async ({ api }) => {
  14 |     const version = await api.get<{ current: string }>("/api/version");
  15 |     expect(version.current).toBeTruthy();
  16 |   });
  17 | 
  18 |   test("API examples endpoint returns 5 examples", async ({ api }) => {
  19 |     const data = await api.get<{ examples: Array<{ slug: string }> }>("/api/examples");
  20 |     expect(data.examples).toHaveLength(5);
  21 |     expect(data.examples[0].slug).toBe("startup");
  22 |   });
  23 | 
  24 |   test("API agents endpoint returns installed agents", async ({ api }) => {
  25 |     await api.installExample("matrix");
  26 |     const agents = await api.get<Array<{ name: string }>>("/api/agents");
  27 |     const names = agents.map((a) => a.name);
> 28 |     expect(names).toContain("Neo");
     |                   ^ Error: expect(received).toContain(expected) // indexOf
  29 |   });
  30 | 
  31 |   test("doctor check passes (Docker running)", async ({ api }) => {
  32 |     const output = api.cli("doctor");
  33 |     expect(output).toContain("Docker");
  34 |     expect(output).toContain("✓");
  35 |   });
  36 | });
  37 | 
```