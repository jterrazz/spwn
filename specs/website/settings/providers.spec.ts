import { button, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('providers', () => {
    test('a host with no credentials lists every provider as not configured', async () => {
        // Given - a spwn home with no keychain and no provider token
        const result = await website.visit('/providers', async (visitor) => {
            await visitor.see(heading('Settings'));
            await visitor.see(button('Docker status: v', { exact: false }));
        });

        // Then - each provider offers to reconnect and explains how
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
              - heading "Settings" [level=1]
              - paragraph: Connect your AI providers to power agents.
              - text: ◆ Anthropic Not configured
              - button "Reconnect"
              - button "How to connect"
              - text: ◎ OpenAI Not configured
              - button "Reconnect"
              - button "How to connect"
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
    });
});
