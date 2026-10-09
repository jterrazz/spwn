import { heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('architect', () => {
    test('an offline architect offers to start, from the page and from the terminal', async () => {
        // Given - a spwn home whose architect never started
        const result = await website.visit('/architect', async (visitor) => {
            await visitor.see(heading('Architect is offline'));
        });

        // Then - the start button and the command that does the same
        expect(result.tree.text).toContain(`  - heading "Architect is offline" [level=2]
  - paragraph: "The Architect runs in the background and manages everything for you: creating agents, spawning worlds, and keeping track of tasks. Start it to begin working."
  - button "Start Architect":
    - img
    - text: Start Architect
  - img
  - text: spwn architect start`);
        expect(result.errors.text).toBe('');
    });
});
