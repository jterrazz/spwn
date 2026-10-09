import { button, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('tool', () => {
    test('a tool page reads its facts and renders its skill file', async () => {
        // Given - the codex runtime, which ships one skill file
        const result = await website.visit('/tools/codex', async (visitor) => {
            await visitor.see(heading('spwn:codex'));
            await visitor.see(button('Docker status: v', { exact: false }));
        });

        // Then - the skill's sections, headings and code rendered from its markdown
        expect(result.tree).toMatch(`
            - link "⬡ spwn":
              - /url: /
            - text: connected
            - button "Search (⌘K)":
              - img
            - list:
              - listitem:
                - button "Architect":
                  - img
                  - text: Architect
              - listitem:
                - button "Settings":
                  - img
                  - text: Settings
            - text: Universe
            - list:
              - listitem:
                - button "Worlds":
                  - img
                  - text: Worlds
              - listitem:
                - button "Agents":
                  - img
                  - text: Agents
              - listitem:
                - button "Tools":
                  - img
                  - text: Tools
              - listitem:
                - button "Organizations":
                  - img
                  - text: Organizations
            - button "matrix stopped · 1 agent"
            - button "startup"
            - list:
              - listitem:
                - button "Home":
                  - img
                  - text: Home
              - listitem:
                - button "Knowledge":
                  - img
                  - text: Knowledge
              - listitem:
                - button "neo":
                  - img
                  - text: neo
            - 'button "Docker status: v{{semver}}"':
              - img
              - text: v{{semver}}
            - link "Docs":
              - /url: https://spwn.sh/docs
              - img
            - link "GitHub":
              - /url: https://github.com/jterrazz/spwn
              - img
            - link "Feedback":
              - /url: https://github.com/jterrazz/spwn/issues/new
              - img
            - button "Toggle theme":
              - img
            - main:
              - button "Back to Tools":
                - img
              - heading "spwn:codex" [level=1]
              - paragraph: OpenAI Codex agent runtime
              - paragraph: Status
              - img
              - text: Available
              - paragraph: Version
              - text: latest
              - paragraph: Skills
              - img
              - text: 1 file
              - paragraph: Verify
              - img
              - text: 1 check
              - paragraph: Provides
              - paragraph: codex CLI + pre-configured workspace trust
              - paragraph: Use when
              - paragraph: You want to use OpenAI models (GPT-5, o3) as the agent runtime
              - paragraph: Dependencies
              - button "spwn:node"
              - paragraph: Verification commands
              - text: $ command -v codex
              - paragraph: Add to world manifest
              - text: "tools: - spwn:codex"
              - paragraph: spwn:node will be installed automatically.
              - heading "Skills" [level=2]
              - paragraph:
                - text: Skills are markdown guides installed at
                - code: /world/skills/codex/
                - text: inside the container. Agents read these to learn how to use the tool.
              - img
              - text: SKILL.md
              - paragraph: Codex is OpenAI's agent runtime - a CLI that executes tasks using GPT models.
              - heading "Usage" [level=3]
              - text: "codex \\"your task here\\" # Interactive mode codex exec \\"your task here\\" # Non-interactive mode codex exec \\"task\\" --full-auto # Full auto with sandboxed writes codex exec \\"task\\" --model gpt-5.4 # Specify model"
              - heading "Configuration" [level=3]
              - paragraph:
                - text: Codex config lives at
                - code: ~/.codex/config.toml
                - text: inside the container.
              - paragraph: Auth tokens are forwarded from the host automatically.
              - heading "Environment" [level=3]
              - text: "- Auth is handled via OAuth tokens (subscription-based, e.g. ChatGPT Plus) - Tokens are mounted from the host at"
              - code: ~/.codex/auth.json
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
    });
});
