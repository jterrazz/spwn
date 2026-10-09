import { button, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('agent', () => {
    test('a stopped agent offers its identity to set up and its world to read', async () => {
        // Given - neo, declared in matrix and never started
        const result = await website.visit('/agents/neo', async (visitor) => {
            await visitor.see(heading("Set up neo's identity"));
            await visitor.see(button('Docker status: v', { exact: false }));
            await visitor.see(button('matrix stopped · 1 agent'));
        });

        // Then - the page lays out the agent's actions, chat and diagnostics
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
              - heading "neo" [level=1]
              - paragraph: · worker
              - button "Dream":
                - img
                - text: Dream
              - button "Fork":
                - img
                - text: Fork
              - button "Export":
                - img
                - text: Export
              - button "Deploy":
                - img
                - text: Deploy
              - button "Delete":
                - img
                - text: Delete
              - heading "Set up neo's identity" [level=3]
              - paragraph: Give this agent a purpose so it knows what to focus on.
              - button:
                - img
              - text: Purpose
              - textbox "What is this agent's purpose?"
              - text: Profile
              - textbox "Describe their profile..."
              - text: Traits
              - textbox "e.g. curious, creative, diligent"
              - paragraph: Press Enter in any field to save. Fill purpose to dismiss this banner.
              - button "Chat":
                - img
                - text: Chat
              - button "Profile"
              - button "Files (0)"
              - img
              - paragraph: Chat with neo
              - paragraph: Send messages directly to this agent in real-time
              - button "What are you working on?"
              - button "Show me the project structure"
              - button "Run the tests"
              - textbox "Message neo..."
              - button "Send" [disabled]:
                - img
              - paragraph: Diagnostics
              - paragraph: Identity
              - paragraph: Name
              - text: neo
              - paragraph: Role
              - text: worker
              - paragraph: Status
              - text: stopped
              - paragraph: Runtime
              - paragraph: Metrics
              - paragraph: Files
              - paragraph: "0"
              - paragraph: Journal
              - paragraph: "0"
              - paragraph: Skills
              - paragraph: "0"
              - paragraph: Traits
              - paragraph: "0"
              - paragraph: World
              - paragraph: Name
              - text: matrix
              - paragraph: Status
              - text: stopped
              - paragraph: Agents
              - text: "1"
              - paragraph: Workspaces
              - text: "1"
            - button "Open glossary":
              - img
            - button "Architect offline":
              - img
              - text: Architect offline
            - alert`);
        expect(result.errors.text).toBe('');
    });
});
