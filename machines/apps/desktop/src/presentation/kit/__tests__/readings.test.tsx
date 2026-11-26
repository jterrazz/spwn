import { button, component, content } from '@jterrazz/test';
import { describe, expect, test, vi } from 'vitest';

import { EngineError } from '../../../domain/engine-error';
import { Unanswered } from '../readings';

const REFUSED = new EngineError({
    fix: { Args: null, Call: '', Command: 'station engine start', Confirm: '', On: '', Path: '' },
    hint: 'the engine of this Mac is stopped',
    message: 'no engine answers',
    type: 'engine-stopped',
});

describe('readings', () => {
    test('its fix command is never drawn, and its Copy the command button copies it', async () => {
        // Given - a refusal whose fix is a command, and a clipboard that takes it
        const written = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue();
        await component.render(<Unanswered error={REFUSED} size="tab" />, async (visitor) => {
            await visitor.see(content('the engine of this Mac is stopped', { exact: false }));
            await visitor.gone(content('station engine start'));

            // When - its Copy the command is clicked
            await visitor.click(button('Copy the command'));

            // Then - the button says it copied
            await visitor.see(button('Copied'));
        });

        // Then - the clipboard holds the command alone
        expect(written).toHaveBeenCalledWith('station engine start');
    });

    test('a hint that is a command is withheld, and the words around it stay', async () => {
        // Given - a refusal whose hint is a command line
        const refused = new EngineError({
            hint: 'station machines start api',
            message: 'api is stopped',
            type: 'refused',
        });

        // When - it is drawn
        const result = await component.render(
            <Unanswered error={refused} size="tab" />,
            async (visitor) => {
                await visitor.see(content('api is stopped.'));
            },
        );

        // Then - the command is nowhere on the page
        expect(result.tree.text).not.toContain('station');
    });
});
