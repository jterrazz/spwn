import { button, content, field } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('organizations', () => {
    test('the built-in organization is drawn as a chart and cannot be deleted', async () => {
        // Given - a host with only the built-in organization
        const result = await website.visit('/organizations', async (visitor) => {
            await visitor.see(content('Built-in three-tier organization'));
            await visitor.see(content('reports to manager'));
            await visitor.see(button('Docker status: v', { exact: false }));
        });

        // Then - its three levels are on the chart, with no button to delete it
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
              - heading "Organizations" [level=1]
              - paragraph: Define team structures and roles for multi-agent worlds.
              - button "New Organization":
                - img
                - text: New Organization
              - paragraph: Default
              - application:
                - img:
                  - group "Edge from chief to manager"
                - img:
                  - group "Edge from manager to worker"
                - group:
                  - paragraph: chief
                  - text: Level 0 max 1 delegate review orchestrate
                  - paragraph: commands manager, worker
                - group:
                  - paragraph: manager
                  - text: Level 1 delegate review execute
                  - paragraph: reports to chief
                  - paragraph: commands worker
                - group:
                  - paragraph: worker
                  - text: Level 2 execute report
                  - paragraph: reports to manager
              - paragraph: Details
              - paragraph: Slug
              - text: default
              - paragraph: Description
              - text: Built-in three-tier organization
              - paragraph: Roles
              - text: "3"
              - paragraph: Max Depth
              - text: "2"
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
    });

    test('an organization declared through the form joins the page, then is deleted', async () => {
        // Given - the built-in organization alone
        const result = await website.visit('/organizations', async (visitor) => {
            await visitor.see(content('Built-in three-tier organization'));

            // When - the visitor declares a crew of one captain, then deletes it
            await visitor.click(button('New Organization'));
            await visitor.fill(field('Name'), 'Crew');
            await visitor.fill(field('Role 1 name'), 'captain');
            await visitor.click(button('Create'));
            await visitor.see(content('captain'));
            await visitor.click(button('Delete Crew'));
            await visitor.click(button('Delete'));
            await visitor.gone(button('Delete Crew'));
            await visitor.see(content('reports to manager'));
            await visitor.see(button('Docker status: v', { exact: false }));
        });

        // Then - only the built-in organization remains
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
              - heading "Organizations" [level=1]
              - paragraph: Define team structures and roles for multi-agent worlds.
              - button "New Organization":
                - img
                - text: New Organization
              - paragraph: Default
              - application:
                - img:
                  - group "Edge from chief to manager"
                - img:
                  - group "Edge from manager to worker"
                - group:
                  - paragraph: chief
                  - text: Level 0 max 1 delegate review orchestrate
                  - paragraph: commands manager, worker
                - group:
                  - paragraph: manager
                  - text: Level 1 delegate review execute
                  - paragraph: reports to chief
                  - paragraph: commands worker
                - group:
                  - paragraph: worker
                  - text: Level 2 execute report
                  - paragraph: reports to manager
              - paragraph: Details
              - paragraph: Slug
              - text: default
              - paragraph: Description
              - text: Built-in three-tier organization
              - paragraph: Roles
              - text: "3"
              - paragraph: Max Depth
              - text: "2"
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
    });
});
