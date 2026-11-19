import { button, content, dialog, disabled, field, heading, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const TWO_MACS = '/?world=two-macs&page=home';

describe('new machine', () => {
    test('H7 a machine launched with what it is for opens running, and says it', async () => {
        // Given - the New machine sheet opened from Home over two-macs, on mac-mini
        const result = await website.visit(TWO_MACS, async (visitor) => {
            await visitor.click(button('New machine'));
            const sheet = dialog('New machine on mac-mini');

            // When - what it is for is said, and linux-2 is launched and confirmed
            await visitor.fill(within(sheet, field('What is it for?')), 'Lot shell');
            await visitor.click(within(sheet, button('Launch')));
            await visitor.click(within(dialog('Launch linux-2?'), button('Launch')));

            // Then - the sheet hands over to linux-2's page, running, with what it is for
            await visitor.gone(sheet);
            await visitor.see(heading('linux-2 running'));
            await visitor.see(content('Lot shell'));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'H7 waits for reads made while a create runs, so its line reads creating before running (a gesture is recorded with the reads after its end)',
    );

    test('H8 a preset sets the memory, the figure and what it leaves follow, and no size above what is free is offered', async () => {
        // Given - the New machine sheet opened from Home over two-macs, on the Mac with 36 GB free
        const result = await website.visit(TWO_MACS, async (visitor) => {
            await visitor.click(button('New machine'));
            const sheet = dialog('New machine on mac-mini');
            await visitor.see(within(sheet, content('leaves 28 GB free on mac-mini')));

            // When - the 12 GB preset is pressed
            await visitor.click(within(sheet, button('12 GB · gate')));

            // Then - the figure and what it leaves follow
            await visitor.see(within(sheet, content('leaves 24 GB free on mac-mini')));

            // When - the Mac with 4 GB free is chosen
            await visitor.click(within(sheet, button('macbook-pro')));

            // Then - the sheet follows that Mac, and the sizes above its room are refused
            const onIt = dialog('New machine on macbook-pro');
            await visitor.see(within(onIt, content('leaves 0 GB free on this Mac')));
            const eight = within(onIt, button('8 GB · no room'));
            const twelve = within(onIt, button('12 GB · no room'));
            await visitor.see(disabled(eight));
            await visitor.see(disabled(twelve));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo('H8 waits for a verb that drags across the free squares of a grid');
});
