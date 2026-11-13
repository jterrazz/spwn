import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { StatePill } from '../state-pill';

describe('StatePill', () => {
    test('shows its word', async () => {
        // Given - a pill for an exposed port
        const result = await component.render(<StatePill tone="bad" word="Exposed" />);

        // Then - the word is what it says
        expect(result.content.text).toBe('Exposed');
        expect(result.errors.text).toBe('');
    });

    test('refuses a state with no word', async () => {
        // Given - a pill whose word is blank
        const render = component.render(<StatePill tone="ok" word=" " />);

        // Then - the render fails on the rule
        await expect(render).rejects.toThrow(
            'A state pill carries its word: a colour alone is no state.',
        );
    });
});
