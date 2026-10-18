import { button, content, field, heading, main, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const settled = button('Docker status: v', { exact: false });

describe('landing', () => {
    test('draws every declared world as a planet, none of them running', async () => {
        // Given - a project declaring the matrix and startup worlds, neither started
        const result = await website.visit('/', async (visitor) => {
            // When - the page has read the API and the Docker daemon
            await visitor.see(button('2 worlds'));
            await visitor.see(settled);
            await visitor.see(button('matrix stopped · 1 agent'));
        });

        // Then - the outline carries both planets and the counters, and the console stayed clean
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
        expect(result.errors.text).toBe('');
    });

    test('selecting a world lays out its roster', async () => {
        // Given - the landing page, read
        const result = await website.visit('/', async (visitor) => {
            await visitor.see(button('2 worlds'));
            await visitor.see(settled);
            await visitor.see(button('matrix stopped · 1 agent'));

            // When - the visitor walks onto the first planet
            await visitor.press('ArrowRight');
            await visitor.see(heading('matrix'));
        });

        // Then - the panel names the world, its state and its one agent
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
              - heading "matrix" [level=3]
              - paragraph: matrix · stopped
              - button "Close":
                - img
              - paragraph: Uptime
              - paragraph: "-"
              - paragraph: Agents
              - paragraph: "1"
              - paragraph: Workspaces
              - paragraph: "1"
              - paragraph: Alive
              - text: 0%
              - paragraph: Agents
              - link "neo stopped":
                - /url: /agents/neo?world=matrix
              - paragraph: Workspaces
              - text: . .
              - button "Enter World":
                - text: Enter World
                - img
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
    });

    test('the New World button opens the spawn form, previewing the command it runs', async () => {
        // Given - the landing page, read
        const result = await website.visit('/', async (visitor) => {
            await visitor.see(button('2 worlds'));
            await visitor.see(settled);

            // When - the visitor asks for a new world and picks neo
            await visitor.click(within(main(), button('New World')));
            await visitor.check(field('neo'));
            await visitor.see(content('→ Deploys 1 agent: neo', { exact: false }));
        });

        // Then - the preview states the command, its default workspace mounted
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
              - button "Close"
              - heading "New World" [level=2]
              - paragraph: Create a new isolated world for your agent
              - button "Close":
                - img
              - text: Name (optional)
              - textbox "Name (optional)":
                - /placeholder: My Project
              - text: Agents (1 selected)
              - button "Clear"
              - list:
                - listitem:
                  - checkbox "analyst"
                  - text: analyst
                - listitem:
                  - checkbox "ceo"
                  - text: ceo
                - listitem:
                  - checkbox "neo" [checked]
                  - text: neo
              - textbox "New agent name":
                - /placeholder: New agent name...
              - button "+ Add" [disabled]
              - text: Workspaces
              - button "+ Add"
              - textbox "Workspace name":
                - /placeholder: name
                - text: default
              - textbox "Workspace path":
                - /placeholder: /tmp/spwn-neo-...
              - button "ro"
              - button "Remove the workspace":
                - img
              - text: Config
              - combobox "Config":
                - option "default" [selected]
                - option "backend"
                - option "frontend"
                - option "fullstack"
                - option "devops"
                - option "minimal"
              - text: Agent Role
              - combobox "Agent Role":
                - option "Chief"
                - option "Manager"
                - option "Worker" [selected]
              - paragraph: Preview
              - text: spwn up -a neo --role worker --config default -w default=/tmp/spwn-neo-...
              - paragraph: → Creates isolated Docker container
              - paragraph: "→ Deploys 1 agent: neo"
              - paragraph: "→ default: /tmp/spwn-neo-..."
              - button "New World":
                - img
                - text: New World
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
    });
});
