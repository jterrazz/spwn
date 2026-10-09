import { button, dialog, option } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

// invariant: the shortcut is bound by an effect, so a visitor reaches for it only once the client has read the Docker daemon
const settled = button('Docker status: v', { exact: false });

describe('palette', () => {
    test('opens on the command shortcut', async () => {
        // Given - the landing page, hydrated
        const result = await website.visit('/', async (visitor) => {
            await visitor.see(settled);

            // When - the visitor presses the shortcut
            await visitor.press('Meta+k');
            await visitor.see(dialog('Command Palette'));
            await visitor.see(option('matrix 1 agents · stopped'));
        });

        // Then - the palette offers to search the commands, the worlds and the agents
        expect(result.tree).toMatch(`
            - dialog "Command Palette":
              - heading "Command Palette" [level=2]
              - paragraph: Search for a command to run...
              - combobox [expanded]
              - img
              - listbox "Suggestions":
                - group "Worlds":
                  - option "matrix 1 agents · stopped":
                    - img
                    - text: matrix 1 agents · stopped
                    - img
                  - option "startup 2 agents · stopped":
                    - img
                    - text: startup 2 agents · stopped
                    - img
                - separator
                - group "Agents":
                  - option "analyst in startup":
                    - img
                    - text: analyst in startup
                    - img
                  - option "ceo in startup":
                    - img
                    - text: ceo in startup
                    - img
                  - option "neo in matrix":
                    - img
                    - text: neo in matrix
                    - img
                - separator
                - group "Navigation":
                  - option "Go to Dashboard" [selected]:
                    - img
                    - text: Go to Dashboard
                    - img
                  - option "Go to Architect":
                    - img
                    - text: Go to Architect
                    - img
                - separator
                - group "Actions":
                  - option "Spawn World ⌘N":
                    - img
                    - text: Spawn World ⌘N
                    - img
                  - option "Create Agent":
                    - img
                    - text: Create Agent
                    - img`);
    });

    test('closes on escape', async () => {
        // Given - the palette, open
        const result = await website.visit('/', async (visitor) => {
            await visitor.see(settled);
            await visitor.press('Meta+k');
            await visitor.see(dialog('Command Palette'));

            // When - the visitor dismisses it
            await visitor.press('Escape');
            await visitor.gone(dialog('Command Palette'));
        });

        // Then - nothing of the palette stays on the page
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
              - heading "Worlds" [level=1]
              - paragraph: Isolated environments where your agents live and work.
              - button "2 worlds":
                - img
                - text: 2 worlds
              - button "0 alive":
                - img
                - text: 0 alive
              - button "0 sleeping":
                - img
                - text: 0 sleeping
              - button "New World":
                - img
                - text: New World
              - button "matrix"
              - button "startup"
              - button
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
    });
});
