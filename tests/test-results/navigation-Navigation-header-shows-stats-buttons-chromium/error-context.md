# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: navigation.spec.ts >> Navigation >> header shows stats buttons
- Location: ui/specs/navigation.spec.ts:36:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByRole('button', { name: /WORLDS/ })
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for getByRole('button', { name: /WORLDS/ })

```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
  - generic [ref=e3]:
    - generic [ref=e4]:
      - generic [ref=e9]:
        - generic [ref=e11]:
          - generic [ref=e12]:
            - link "⬡ spwn" [ref=e13] [cursor=pointer]:
              - /url: /
              - generic [ref=e14]: ⬡
              - generic [ref=e15]: spwn
            - generic [ref=e16]: connected
          - button "Search (⌘K)" [ref=e17] [cursor=pointer]:
            - img [ref=e18]
        - generic [ref=e21]:
          - list [ref=e23]:
            - listitem [ref=e24]:
              - button "Architect" [ref=e25] [cursor=pointer]:
                - img [ref=e26]
                - generic [ref=e28]: Architect
            - listitem [ref=e29]:
              - button "Settings" [ref=e30] [cursor=pointer]:
                - img [ref=e31]
                - generic [ref=e33]: Settings
          - generic [ref=e34]:
            - generic [ref=e35]: Universe
            - list [ref=e36]:
              - listitem [ref=e37]:
                - button "Worlds" [ref=e38] [cursor=pointer]:
                  - img [ref=e39]
                  - generic [ref=e41]: Worlds
              - listitem [ref=e42]:
                - button "Agents" [ref=e43] [cursor=pointer]:
                  - img [ref=e44]
                  - generic [ref=e47]: Agents
              - listitem [ref=e48]:
                - button "Tools" [ref=e49] [cursor=pointer]:
                  - img [ref=e50]
                  - generic [ref=e52]: Tools
              - listitem [ref=e53]:
                - button "Organizations" [ref=e54] [cursor=pointer]:
                  - img [ref=e55]
                  - generic [ref=e57]: Organizations
          - generic [ref=e59]:
            - paragraph [ref=e60]: Getting started
            - generic [ref=e61]:
              - paragraph [ref=e62]:
                - text: 1. Go to
                - button "Settings" [ref=e63] [cursor=pointer]
                - text: and connect a provider
              - paragraph [ref=e64]:
                - text: 2. Create an
                - button "Agent" [ref=e65] [cursor=pointer]
              - paragraph [ref=e66]:
                - text: 3. Spawn a
                - button "World" [ref=e67] [cursor=pointer]
          - paragraph [ref=e69]: No worlds running
        - generic [ref=e70]:
          - 'button "Docker status: v28.5.2" [ref=e72] [cursor=pointer]':
            - img [ref=e73]
            - generic [ref=e85]: v28.5.2
          - generic [ref=e86]:
            - link "Docs" [ref=e87] [cursor=pointer]:
              - /url: https://spwn.sh/docs
              - img [ref=e88]
            - link "GitHub" [ref=e90] [cursor=pointer]:
              - /url: https://github.com/jterrazz/spwn
              - img [ref=e91]
            - link "Feedback" [ref=e93] [cursor=pointer]:
              - /url: https://github.com/jterrazz/spwn/issues/new
              - img [ref=e94]
            - button "Toggle theme" [ref=e96] [cursor=pointer]:
              - img [ref=e97]
      - main [ref=e107]:
        - generic [ref=e108]:
          - generic [ref=e109]:
            - generic [ref=e111]:
              - heading "Worlds" [level=1] [ref=e112]
              - paragraph [ref=e113]: Isolated environments where your agents live and work.
            - generic [ref=e115]:
              - generic [ref=e116]:
                - button "0 worlds" [ref=e117] [cursor=pointer]:
                  - img [ref=e120]
                  - generic [ref=e130]:
                    - generic [ref=e131]: "0"
                    - generic [ref=e132]: worlds
                - button "0 alive" [ref=e133] [cursor=pointer]:
                  - img [ref=e136]
                  - generic [ref=e138]:
                    - generic [ref=e139]: "0"
                    - generic [ref=e140]: alive
                - button "0 sleeping" [ref=e141] [cursor=pointer]:
                  - img [ref=e144]
                  - generic [ref=e146]:
                    - generic [ref=e147]: "0"
                    - generic [ref=e148]: sleeping
              - button "New World" [ref=e149] [cursor=pointer]:
                - generic:
                  - img
                - generic [ref=e150]: New World
          - generic [ref=e152]:
            - generic [ref=e153]:
              - generic [ref=e154]:
                - img [ref=e155]
                - text: Start from a template
              - heading "Give your agents a world to work in" [level=2] [ref=e157]
              - paragraph [ref=e158]: You have 11 agents installed. Pick a template to put one to work, or build your own world from scratch.
            - generic [ref=e159]:
              - generic [ref=e160]:
                - generic [ref=e161]:
                  - img [ref=e164]
                  - generic [ref=e168]:
                    - heading "Startup" [level=3] [ref=e169]
                    - paragraph [ref=e170]: One world, one team, three agents
                - paragraph [ref=e171]: A tiny startup with a CEO, a devops engineer, and a research analyst all working together in a single world. The CEO decides what ships, devops keeps the pipeline green, and the analyst explores new ideas.
                - generic [ref=e172]:
                  - generic [ref=e173]:
                    - img [ref=e174]
                    - text: ceo
                  - generic [ref=e177]:
                    - img [ref=e178]
                    - text: devops
                  - generic [ref=e181]:
                    - img [ref=e182]
                    - text: analyst
                - code [ref=e186]: $ spwn up -c startup --leader ceo --agent devops --agent analyst
                - button "Install & spawn" [ref=e188] [cursor=pointer]:
                  - img [ref=e189]
                  - text: Install & spawn
                  - img [ref=e193]
              - generic [ref=e196]:
                - generic [ref=e197]:
                  - img [ref=e200]
                  - generic [ref=e204]:
                    - heading "The Matrix" [level=3] [ref=e205]
                    - paragraph [ref=e206]: A sandbox with Neo — interactive exploration
                - paragraph [ref=e207]: A clean room. One agent named Neo. No goals, no tasks, no backlog — just a place to poke the system, try tools, and see what emerges.
                - generic [ref=e209]:
                  - img [ref=e210]
                  - text: neo
                - code [ref=e214]: $ spwn up -c matrix --agent neo
                - button "Install & spawn" [ref=e215] [cursor=pointer]:
                  - img [ref=e216]
                  - text: Install & spawn
                  - img [ref=e220]
              - generic [ref=e223]:
                - generic [ref=e224]:
                  - img [ref=e227]
                  - generic [ref=e230]:
                    - heading "Paperclip Factory" [level=3] [ref=e231]
                    - paragraph [ref=e232]: Your single-agent automation workshop
                - paragraph [ref=e233]: One tireless worker. A world built for loops, scripts, and scheduled work. Clippy never sleeps — give it a directory full of things to process and it will keep maximizing whatever you tell it to maximize.
                - generic [ref=e235]:
                  - img [ref=e236]
                  - text: clippy
                - code [ref=e240]: $ spwn up -c paperclip-factory --agent clippy
                - button "Install & spawn" [ref=e242] [cursor=pointer]:
                  - img [ref=e243]
                  - text: Install & spawn
                  - img [ref=e247]
              - generic [ref=e250]:
                - generic [ref=e251]:
                  - img [ref=e254]
                  - generic [ref=e256]:
                    - heading "Research Lab" [level=3] [ref=e257]
                    - paragraph [ref=e258]: A lab notebook and a scientist you can fork
                - paragraph [ref=e259]: "A patient, methodical agent named Curie. Curie keeps a real lab notebook — hypotheses, methods, observations, conclusions — and writes playbooks as she figures things out. Designed to showcase the \"same brain, new soul\" pattern: once Curie has learned enough, fork her into Darwin and watch him specialize differently."
                - generic [ref=e261]:
                  - img [ref=e262]
                  - text: curie
                - code [ref=e266]: $ spwn up -c research-lab --agent curie
                - button "Install & spawn" [ref=e267] [cursor=pointer]:
                  - img [ref=e268]
                  - text: Install & spawn
                  - img [ref=e272]
              - generic [ref=e275]:
                - generic [ref=e276]:
                  - img [ref=e279]
                  - generic [ref=e284]:
                    - heading "Macrohard" [level=3] [ref=e285]
                    - paragraph [ref=e286]: Your three-agent software company in a box
                - paragraph [ref=e287]: A tiny company with a chief and two developers. Ballmer assigns work, Gates and Nadella build it. The three agents live in the same world and communicate through their per-world inboxes.
                - generic [ref=e288]:
                  - generic [ref=e289]:
                    - img [ref=e290]
                    - text: ballmer
                  - generic [ref=e293]:
                    - img [ref=e294]
                    - text: gates
                  - generic [ref=e297]:
                    - img [ref=e298]
                    - text: nadella
                - code [ref=e302]: $ spwn up -c macrohard --agents ballmer,gates,nadella
                - button "Install & spawn" [ref=e303] [cursor=pointer]:
                  - img [ref=e304]
                  - text: Install & spawn
                  - img [ref=e308]
            - generic [ref=e311]:
              - button "Or build your own world from scratch" [ref=e312] [cursor=pointer]:
                - img [ref=e313]
                - text: Or build your own world from scratch
              - generic [ref=e317]:
                - img [ref=e318]
                - generic [ref=e321]: spwn example list
      - button "Open glossary" [ref=e322] [cursor=pointer]:
        - img [ref=e323]
    - generic [ref=e327]:
      - button "Architect alive" [ref=e328] [cursor=pointer]:
        - img [ref=e331]
      - textbox "Ask the Architect..." [ref=e333]
    - generic [ref=e334]:
      - heading "Command Palette" [level=2] [ref=e335]
      - paragraph [ref=e336]: Search for a command to run...
  - button "Open Next.js Dev Tools" [ref=e342] [cursor=pointer]:
    - img [ref=e343]
  - alert [ref=e346]
```

# Test source

```ts
  1  | import { test, expect } from "../fixtures/app";
  2  | 
  3  | test.describe("Navigation", () => {
  4  |   test.beforeEach(async ({ page }) => {
  5  |     await page.goto("/");
  6  |     await page.waitForTimeout(2000);
  7  |   });
  8  | 
  9  |   test("sidebar navigation changes page content", async ({ page }) => {
  10 |     // Click Agents — content should change
  11 |     await page.getByRole("button", { name: "Agents", exact: true }).first().click();
  12 |     await page.waitForTimeout(1000);
  13 |     await expect(page.getByRole("heading", { name: "Agents" })).toBeVisible({ timeout: 5000 });
  14 | 
  15 |     // Click Tools
  16 |     await page.getByRole("button", { name: "Tools", exact: true }).first().click();
  17 |     await page.waitForTimeout(1000);
  18 |     await expect(page.getByRole("heading", { name: "Tools" })).toBeVisible({ timeout: 5000 });
  19 | 
  20 |     // Click back to Worlds
  21 |     await page.getByRole("button", { name: "Worlds", exact: true }).first().click();
  22 |     await page.waitForTimeout(1000);
  23 |     await expect(page.getByRole("heading", { name: "Worlds", level: 1 })).toBeVisible({ timeout: 5000 });
  24 |   });
  25 | 
  26 |   test("command palette opens with Cmd+K", async ({ page }) => {
  27 |     await page.keyboard.press("Meta+k");
  28 |     await page.waitForTimeout(500);
  29 | 
  30 |     await expect(page.getByText(/Search for a command/i)).toBeVisible({ timeout: 3000 });
  31 | 
  32 |     await page.keyboard.press("Escape");
  33 |     await page.waitForTimeout(300);
  34 |   });
  35 | 
  36 |   test("header shows stats buttons", async ({ page }) => {
> 37 |     await expect(page.getByRole("button", { name: /WORLDS/ })).toBeVisible({ timeout: 5000 });
     |                                                                ^ Error: expect(locator).toBeVisible() failed
  38 |   });
  39 | 
  40 |   test("Docker version is visible in sidebar", async ({ page }) => {
  41 |     await expect(page.getByRole("button", { name: /Docker status/ })).toBeVisible({ timeout: 5000 });
  42 |   });
  43 | });
  44 | 
```