import { button, heading, link } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('world', () => {
    test('a declared world lays out its state and its one agent', async () => {
        // Given - matrix, declared with neo and never started
        const result = await website.visit('/world/matrix', async (visitor) => {
            await visitor.see(heading('matrix'));
            await visitor.see(link('neo worker stopped', { exact: false }));
            await visitor.see(button('Docker status: v', { exact: false }));
            await visitor.see(button('matrix stopped · 1 agent'));
        });

        // Then - the page reads its counters and its roster
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
            - 'button "Docker status: v29.4.0"':
              - img
              - text: v29.4.0
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
              - heading "matrix" [level=1]
              - paragraph: matrix · .
              - button "Snapshot":
                - img
                - text: Snapshot
              - button "Logs":
                - img
                - text: Logs
              - button "Snapshots":
                - img
                - text: Snapshots
              - button "Rename":
                - img
                - text: Rename
              - button "Destroy":
                - img
                - text: Destroy
              - paragraph: Status
              - paragraph: stopped
              - paragraph: Agents
              - paragraph: "1"
              - paragraph: Config
              - paragraph: matrix
              - paragraph: Uptime
              - paragraph: "-"
              - paragraph: Agents
              - button "+ Deploy"
              - paragraph: Name
              - paragraph: Role
              - paragraph: Status
              - link "neo worker stopped":
                - /url: /agents/neo?world=matrix
                - text: neo worker stopped
                - img
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
    });

    test('the logs panel opens beside the world', async () => {
        // Given - the world page
        const result = await website.visit('/world/matrix', async (visitor) => {
            await visitor.see(heading('matrix'));

            // When - the visitor opens the logs
            await visitor.click(button('Logs'));
            await visitor.see(heading('logs'));
        });

        // Then - a world that never ran has no activity to show
        expect(result.content.text).toContain('No logs available');
    });
});
