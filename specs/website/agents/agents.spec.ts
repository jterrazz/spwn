import { button, heading, link } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('agents', () => {
    test('lists every agent of the project with the world it is declared in', async () => {
        // Given - a project declaring neo in matrix, ceo and analyst in startup
        const result = await website.visit('/agents', async (visitor) => {
            await visitor.see(link('neo worker stopped matrix'));
            await visitor.see(button('Docker status: v', { exact: false }));
            await visitor.see(button('matrix stopped · 1 agent'));
        });

        // Then - the roster reads one row per agent, by name
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
              - heading "Agents" [level=1]
              - paragraph: Persistent identities that remember across sessions and worlds.
              - button "Search":
                - img
              - button "New Team":
                - img
                - text: New Team
              - button "New Agent":
                - img
                - text: New Agent
              - button "all 3"
              - button "deployed 3"
              - button "limbo 0"
              - paragraph: No team
              - text: "3"
              - paragraph: Name
              - paragraph: Role
              - paragraph: Status
              - paragraph: World
              - link "analyst worker stopped startup":
                - /url: /agents/analyst?world=startup
                - text: analyst worker stopped startup
                - img
              - link "ceo worker stopped startup":
                - /url: /agents/ceo?world=startup
                - text: ceo worker stopped startup
                - img
              - link "neo worker stopped matrix":
                - /url: /agents/neo?world=matrix
                - text: neo worker stopped matrix
                - img
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
    });

    test('an agent row opens that agent in its world', async () => {
        // Given - the roster
        const result = await website.visit('/agents', async (visitor) => {
            // When - the visitor opens neo
            await visitor.click(link('neo worker stopped matrix'));
            await visitor.see(heading('neo'));
        });

        // Then - the address names the agent and the world it was opened from
        const address = new URL(result.url);
        expect(`${address.pathname}${address.search}`).toBe('/agents/neo?world=matrix');
    });
});
