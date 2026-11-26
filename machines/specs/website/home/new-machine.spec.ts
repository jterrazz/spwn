import {
    button,
    content,
    dialog,
    disabled,
    field,
    heading,
    main,
    navigation,
    region,
    testId,
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
            await visitor.click(within(main(), button('New machine')));
            const sheet = dialog('New machine on mac-mini');

            // When - what it is for is said, and linux-2 is launched, which asks nothing
            await visitor.fill(within(sheet, field('What is it for?')), 'Lot shell');
            await visitor.click(within(sheet, button('Launch')));
            await visitor.gone(dialog('Launch linux-2?'));

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
                await visitor.click(within(main(), button('New machine')));
                const sheet = dialog('New machine on mac-mini');

                // When - linux-2 is launched for what it is for, which asks nothing
                await visitor.fill(within(sheet, field('What is it for?')), 'Lot shell');
                await visitor.click(within(sheet, button('Launch')));
                await visitor.gone(dialog('Launch linux-2?'));

                // Then - its page reads creating and its step
                await visitor.see(heading('linux-2 creating'));
                await visitor.see(content('Creating: boot 2 of 4'));

                // When - Home is opened while it runs
                await visitor.click(within(navigation('Rail'), button('Home')));

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
            await visitor.click(within(main(), button('New machine')));
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

    test('H8 a drag across the free squares lights the squares taken among those held, the figure and what it leaves follow, and it stops at what is free', async () => {
        // Given - the New machine sheet over two-macs, on the Mac with 36 GB free: its 64 squares are macOS 4, infra 24, the 8 chosen and the 28 free, two rows of 32
        const result = await website.visit(TWO_MACS, async (visitor) => {
            await visitor.click(within(main(), button('New machine')));
            const sheet = dialog('New machine on mac-mini');
            // testId: the memory grid is an img, a role no locator of the visitor names
            const grid = within(sheet, testId('memory-grid'));
            await visitor.see(within(sheet, content('leaves 28 GB free on mac-mini')));

            // When - the pointer goes down on the free square of column 8, row 2, and is released on column 16, the 20th GB
            await visitor.drag(
                grid,
                { x: (8 - 0.5) / 32, y: 0.75 },
                { x: (16 - 0.5) / 32, y: 0.75 },
            );

            // Then - the figure and what it leaves follow
            await visitor.see(within(sheet, content('20GB')));
            await visitor.see(within(sheet, content('leaves 16 GB free on mac-mini')));

            // When - the pointer goes down on the 20th square and is released on the last one, past which nothing is free
            await visitor.drag(
                grid,
                { x: (16 - 0.5) / 32, y: 0.75 },
                { x: (32 - 0.5) / 32, y: 0.75 },
            );

            // Then - the 36 GB it can hold are chosen, no more
            await visitor.see(within(sheet, content('36GB')));
            await visitor.see(within(sheet, content('leaves 0 GB free on mac-mini')));
        });
        expect(result.errors.text).toBe('');

        // Then - the 36 squares taken are lit, among the 28 held by macOS and infra
        expect(result.html.text.match(/data-identity="[^"]+" data-take=/gu)).toHaveLength(36);
    });

    test('H15 a name the file it follows already bears is refused under the Name field, the sheet kept', async () => {
        // Given - the New machine sheet opened from Home over two-macs, on mac-mini, following base-ubuntu
        const result = await website.visit(TWO_MACS, async (visitor) => {
            await visitor.click(within(main(), button('New machine')));
            const sheet = dialog('New machine on mac-mini');

            // When - base-ubuntu is named and launched, which asks nothing
            await visitor.fill(within(sheet, field('Name')), 'base-ubuntu');
            await visitor.click(within(sheet, button('Launch')));
            await visitor.gone(dialog('Launch base-ubuntu?'));

            // Then - the sheet stays, the engine's sentence under the name
            const said = content("base-ubuntu names a Machinefile of base-ubuntu's chain", {
                exact: false,
            });
            await visitor.see(within(region('What it is called'), said));
            await visitor.see(within(sheet, said));
        });
        expect(result.errors.text).toBe('');
    });

    test('H15 a name taken on the Mac is refused under the Name field', async () => {
        // Given - the New machine sheet opened from Home over two-macs, on mac-mini, which holds infra
        const result = await website.visit(TWO_MACS, async (visitor) => {
            await visitor.click(within(main(), button('New machine')));
            const sheet = dialog('New machine on mac-mini');

            // When - infra is named and launched, which asks nothing
            await visitor.fill(within(sheet, field('Name')), 'infra');
            await visitor.click(within(sheet, button('Launch')));
            await visitor.gone(dialog('Launch infra?'));

            // Then - the engine's sentence under the name
            const said = content('"infra" is taken (on mac-mini)', { exact: false });
            await visitor.see(within(region('What it is called'), said));
        });
        expect(result.errors.text).toBe('');
    });
});
