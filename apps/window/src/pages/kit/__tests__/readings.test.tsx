import { button, component, content } from '@jterrazz/test';
import { describe, expect, test, vi } from 'vitest';

import { EngineError } from '../../../engine/engine-error';
import { Unanswered } from '../readings';

const REFUSED = new EngineError({
    fix: { Args: null, Call: '', Command: 'station engine start', Path: '' },
    hint: 'the engine of this Mac is stopped',
    message: 'no engine answers',
    type: 'engine-stopped',
});

describe('readings', () => {
    test('its fix command is copied by its Copy gesture', async () => {
        // Given - a refusal whose fix is a command, and a clipboard that takes it
        const written = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue();
        await component.render(<Unanswered error={REFUSED} size="tab" />, async (visitor) => {
            await visitor.see(content('station engine start'));

            // When - its Copy is clicked
            await visitor.click(button('Copy station engine start'));

            // Then - the button says it copied
            await visitor.see(button('Copied'));
        });

        // Then - the clipboard holds the command alone
        expect(written).toHaveBeenCalledWith('station engine start');
    });

    test('a fix line withheld on screen offers no Copy of its command', async () => {
        // Given - a refusal whose command is plain but whose fix line names a path shaped like a secret
        const withheld = new EngineError({
            fix: {
                Args: null,
                Call: '',
                Command: 'station engine start',
                Path: `x9${'k7'.repeat(20)}`,
            },
            hint: 'the engine of this Mac is stopped',
            message: 'no engine answers',
            type: 'engine-stopped',
        });
        const result = await component.render(
            <Unanswered error={withheld} size="tab" />,
            async (visitor) => {
                await visitor.see(content('Withheld'));
            },
        );

        // Then - the line reads Withheld and nothing offers to copy it
        expect(result.tree.text).not.toContain('Copy');
    });
});
