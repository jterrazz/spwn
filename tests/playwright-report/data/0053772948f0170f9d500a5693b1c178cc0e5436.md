# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: api-health.spec.ts >> API health >> API is connected (not showing offline banner)
- Location: ui/specs/api-health.spec.ts:4:3

# Error details

```
Error: expect(locator).not.toBeVisible() failed

Locator:  getByText('Waiting for Docker')
Expected: not visible
Received: visible
Timeout:  5000ms

Call log:
  - Expect "not toBeVisible" with timeout 5000ms
  - waiting for getByText('Waiting for Docker')
    9 × locator resolved to <h1 class="font-heading text-xl tracking-wide text-foreground/95">Waiting for Docker</h1>
      - unexpected value "visible"

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
          - generic [ref=e31]: Checking every 3s · last check 2s ago
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
  3  | test.describe("API health", () => {
  4  |   test("API is connected (not showing offline banner)", async ({ page }) => {
  5  |     await page.goto("/");
  6  |     await page.waitForTimeout(3000);
  7  | 
  8  |     // Should NOT show the "Waiting for Docker" or "API OFFLINE" screen
> 9  |     await expect(page.getByText("Waiting for Docker")).not.toBeVisible({ timeout: 5000 });
     |                                                            ^ Error: expect(locator).not.toBeVisible() failed
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
  28 |     expect(names).toContain("Neo");
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