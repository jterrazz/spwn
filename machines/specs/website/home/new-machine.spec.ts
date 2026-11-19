import {
    button,
    content,
    dialog,
    disabled,
    field,
    heading,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const TWO_MACS = '/?world=two-macs&page=home';
const machines = region('Machines');

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

    test("H7 while the create runs, its page and its line read creating, its share on its Mac's track", async () => {
        // Given - the New machine sheet opened from Home over two-macs, a create held after its first step
        const result = await website.visit(
            `${TWO_MACS}&held=machines.create-workspace`,
            async (visitor) => {
                await visitor.click(button('New machine'));
                const sheet = dialog('New machine on mac-mini');

                // When - linux-2 is launched for what it is for
                await visitor.fill(within(sheet, field('What is it for?')), 'Lot shell');
                await visitor.click(within(sheet, button('Launch')));
                await visitor.click(within(dialog('Launch linux-2?'), button('Launch')));

                // Then - its page reads creating and its step
                await visitor.see(heading('linux-2 creating'));
                await visitor.see(content('Creating: boot 2 of 4'));

                // When - Home is opened while it runs
                await visitor.click(within(navigation('Sidebar'), button('Home')));

                // Then - its line reads creating, and its 8 GB are gone from what mac-mini has free
                await visitor.see(within(machines, button('linux-2')));
                await visitor.see(within(machines, content('creating')));
                await visitor.see(within(machines, content('28 free')));
            },
        );
        expect(result.errors.text).toBe('');
    });

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

    test.todo(
        "H8 waits for a drag verb of @jterrazz/test's visitor (its verbs are check, click, fill, gone, hover, press, see, select), and for the memory grid to take a drag: MemoryPick draws its squares as one image, with no pointer handler",
    );
});
