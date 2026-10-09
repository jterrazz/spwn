import { button, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { withoutArchitect } from '../../__setup__/architect.js';
import { website } from '../website.specification.js';

describe('architect', () => {
    test('an offline architect offers to start, from the page and from the terminal', async () => {
        // Given - a spwn home whose architect never started
        const result = await withoutArchitect(
            async () =>
                await website.visit('/architect', async (visitor) => {
                    await visitor.see(heading('Architect is offline'));
                    await visitor.see(button('Docker status: v', { exact: false }));
                }),
        );

        // Then - the start button and the command that does the same
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
              - heading "Architect" [level=1]
              - paragraph: Offline
              - button "Start":
                - img
                - text: Start
              - img
              - heading "Architect is offline" [level=2]
              - paragraph: "The Architect runs in the background and manages everything for you: creating agents, spawning worlds, and keeping track of tasks. Start it to begin working."
              - button "Start Architect":
                - img
                - text: Start Architect
              - img
              - text: spwn architect start
            - button "Open glossary":
              - img
            - alert`);
        expect(result.errors.text).toBe('');
    });
});
