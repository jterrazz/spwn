import { button, content } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('knowledge', () => {
    test('a world with no knowledge yet points to the architect', async () => {
        // Given - matrix, whose knowledge folder holds nothing
        const result = await website.visit('/world/matrix/knowledge', async (visitor) => {
            await visitor.see(content('No knowledge files yet'));
            await visitor.see(button('Docker status: v', { exact: false }));
        });

        // Then - the browser says so and offers the way to start
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
              - heading "Knowledge" [level=1]
              - paragraph: Shared knowledge base for this world.
              - button "Search":
                - img
              - img
              - heading "Knowledge" [level=3]
              - text: managed by architect
              - img
              - paragraph: No knowledge files yet
              - paragraph: Talk to the Architect to start building your knowledge base
              - link "Go to Architect →":
                - /url: /architect
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
    });
});
