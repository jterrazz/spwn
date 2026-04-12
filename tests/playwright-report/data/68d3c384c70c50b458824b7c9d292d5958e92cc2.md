# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: worlds.spec.ts >> Worlds page >> sidebar shows navigation items
- Location: ui/specs/worlds.spec.ts:13:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByRole('button', { name: 'Settings' })
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 10000ms
  - waiting for getByRole('button', { name: 'Settings' })

```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
  - generic [ref=e3]:
    - generic [ref=e4]:
      - generic:
        - generic:
          - generic:
            - generic:
              - generic:
                - generic:
                  - generic:
                    - link:
                      - /url: /
                      - generic: ⬡
                      - generic: spwn
                    - generic: disconnected
                  - button:
                    - img
              - generic:
                - generic:
                  - list:
                    - listitem:
                      - button:
                        - img
                        - generic: Architect
                    - listitem:
                      - button:
                        - img
                        - generic: Settings
                - generic:
                  - generic: Universe
                  - list:
                    - listitem:
                      - button:
                        - img
                        - generic: Worlds
                    - listitem:
                      - button:
                        - img
                        - generic: Agents
                    - listitem:
                      - button:
                        - img
                        - generic: Tools
                    - listitem:
                      - button:
                        - img
                        - generic: Organizations
                - generic:
                  - generic:
                    - paragraph: Getting started
                    - generic:
                      - paragraph:
                        - text: 1. Go to
                        - button: Settings
                        - text: and connect a provider
                      - paragraph:
                        - text: 2. Create an
                        - button: Agent
                      - paragraph:
                        - text: 3. Spawn a
                        - button: World
                - generic:
                  - paragraph: No worlds running
              - generic:
                - generic:
                  - button:
                    - img
                    - generic: API offline
                - generic:
                  - link:
                    - /url: https://spwn.sh/docs
                    - img
                  - link:
                    - /url: https://github.com/jterrazz/spwn
                    - img
                  - link:
                    - /url: https://github.com/jterrazz/spwn/issues/new
                    - img
                  - button:
                    - img
      - main [ref=e6]:
        - generic [ref=e9]:
          - img [ref=e14]
          - heading "Waiting for Docker" [level=1] [ref=e23]
          - paragraph [ref=e24]: Start Docker Desktop and spwn will pick it up automatically.
          - button "Retry now" [ref=e27] [cursor=pointer]:
            - img [ref=e28]
            - text: Retry now
          - generic [ref=e31]: Checking every 3s · last check 0s ago
      - button "Open glossary" [ref=e35] [cursor=pointer]:
        - img [ref=e36]
    - button "Architect offline" [ref=e40] [cursor=pointer]:
      - img [ref=e43]
      - generic [ref=e45]: Architect offline
    - generic [ref=e46]:
      - heading "Command Palette" [level=2] [ref=e47]
      - paragraph [ref=e48]: Search for a command to run...
  - button "Open Next.js Dev Tools" [ref=e54] [cursor=pointer]:
    - img [ref=e55]
  - alert [ref=e58]
```

# Test source

```ts
  1  | import { test, expect } from "../fixtures/app";
  2  | 
  3  | test.describe("Worlds page", () => {
  4  |   test.beforeEach(async ({ page }) => {
  5  |     await page.goto("/");
  6  |     await page.waitForTimeout(2000);
  7  |   });
  8  | 
  9  |   test("shows the Worlds heading", async ({ page }) => {
  10 |     await expect(page.getByRole("heading", { name: "Worlds" })).toBeVisible();
  11 |   });
  12 | 
  13 |   test("sidebar shows navigation items", async ({ page }) => {
  14 |     await expect(page.getByRole("button", { name: "Architect" })).toBeVisible();
> 15 |     await expect(page.getByRole("button", { name: "Settings" })).toBeVisible();
     |                                                                  ^ Error: expect(locator).toBeVisible() failed
  16 |     await expect(page.getByRole("button", { name: "Worlds" })).toBeVisible();
  17 |     await expect(page.getByRole("button", { name: "Agents" })).toBeVisible();
  18 |     await expect(page.getByRole("button", { name: "Tools" })).toBeVisible();
  19 |   });
  20 | 
  21 |   test("shows example gallery when no worlds are running", async ({ page, api }) => {
  22 |     // Ensure no worlds running
  23 |     await api.destroyAll();
  24 |     await page.goto("/");
  25 |     await page.waitForTimeout(3000);
  26 | 
  27 |     // Should show the gallery with templates
  28 |     await expect(page.getByText("Pick a template")).toBeVisible({ timeout: 10_000 });
  29 |     await expect(page.getByText("Startup")).toBeVisible();
  30 |     await expect(page.getByText("The Matrix")).toBeVisible();
  31 |   });
  32 | 
  33 |   test("shows planets when worlds exist", async ({ page, api }) => {
  34 |     // Spawn a world via API
  35 |     await api.installExample("matrix");
  36 |     await api.spawnWorld("matrix", "Neo");
  37 |     await page.goto("/");
  38 |     await page.waitForTimeout(3000);
  39 | 
  40 |     // Should show at least one planet with the "+" new world button
  41 |     await expect(page.getByText("New World")).toBeVisible({ timeout: 10_000 });
  42 |   });
  43 | 
  44 |   test("selecting a planet shows the detail panel", async ({ page, api }) => {
  45 |     await api.installExample("matrix");
  46 |     const result = await api.spawnWorld("matrix", "Neo");
  47 |     await page.goto("/");
  48 |     await page.waitForTimeout(3000);
  49 | 
  50 |     // Click the planet — the sidebar should show world details
  51 |     // Arrow keys navigate, or click the planet label
  52 |     await page.keyboard.press("ArrowRight");
  53 |     await page.waitForTimeout(1500);
  54 | 
  55 |     // Detail panel should show agent name
  56 |     await expect(page.getByText("Neo")).toBeVisible({ timeout: 5000 });
  57 |     await expect(page.getByText(/Enter World/)).toBeVisible();
  58 |   });
  59 | 
  60 |   test("destroying a world removes it from the list", async ({ page, api }) => {
  61 |     await api.installExample("matrix");
  62 |     const result = await api.spawnWorld("matrix", "Neo");
  63 |     const worldId = result.Universe.id;
  64 | 
  65 |     await page.goto("/");
  66 |     await page.waitForTimeout(3000);
  67 | 
  68 |     // Destroy via API
  69 |     await api.destroyWorld(worldId);
  70 |     await page.waitForTimeout(6000); // wait for poll
  71 | 
  72 |     // Should show empty state or gallery
  73 |     await expect(page.getByText(/Pick a template|Give your agents/)).toBeVisible({ timeout: 10_000 });
  74 |   });
  75 | });
  76 | 
```