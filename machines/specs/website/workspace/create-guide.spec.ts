import { banner, button, content, dialog, field, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

type Visitor = Parameters<NonNullable<Parameters<typeof website.visit>[1]>>[0];

const HOME = '/?world=create-held&page=home';
const guide = dialog('New machine linux-2');

async function launched(visitor: Visitor) {
    await visitor.click(within(banner(), button('New machine')));
    await visitor.click(within(region('On'), button('macbook-pro')));
    const sheet = dialog('New machine on macbook-pro');
    await visitor.fill(within(sheet, field('What is it for?')), 'Lot shell');
    await visitor.fill(within(sheet, field('Name')), 'linux-2');
    await visitor.click(within(sheet, button('ephemeral')));
    await visitor.click(within(sheet, button('Launch')));
    await visitor.click(within(dialog('Launch linux-2?'), button('Launch')));
    await visitor.gone(sheet);
}

describe('create-guide', () => {
    test('W33 a create opens its Guide over Home, Hide leaves the act running, and Show progress opens the same Guide', async () => {
        // Given - Home over two-macs, a create held at its boot
        const result = await website.visit(
            `${HOME}&held=machines.create-workspace`,
            async (visitor) => {
                // When - linux-2 is launched
                await launched(visitor);

                // Then - its Guide names the step under way and its last line
                await visitor.see(within(guide, heading('Boot now Network next')));
                await visitor.see(within(guide, content('Running')));

                // When - the Guide is hidden
                await visitor.click(within(guide, button('Hide')));

                // Then - its page offers it again, and opens the same act
                await visitor.see(heading('linux-2 creating'));
                await visitor.click(button('Show progress'));
                await visitor.see(within(guide, heading('Boot now Network next')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W34 a create made says the machine is ready, and its action opens the machine', async () => {
        // Given - Home over two-macs
        const result = await website.visit(HOME, async (visitor) => {
            // When - linux-2 is launched and made
            await launched(visitor);

            // Then - its Guide says it is ready, and Open linux-2 opens its page
            await visitor.see(within(guide, heading('linux-2 is ready')));
            await visitor.click(within(guide, button('Open linux-2')));
            await visitor.see(heading('linux-2 running'));
        });
        expect(result.errors.text).toBe('');
    });

    test('W35 a create that failed at its boot says so with what was kept, and Try again starts it', async () => {
        // Given - Home over create-failing, where the hypervisor refuses linux-2's boot
        const result = await website.visit('/?world=create-failing&page=home', async (visitor) => {
            // When - linux-2 is launched
            await launched(visitor);

            // Then - its Guide reads failed at Boot, with its two repairs
            await visitor.see(within(guide, heading('Boot failed')));
            await visitor.see(within(guide, button('Delete linux-2')));

            // When - Try again is pressed
            await visitor.click(within(guide, button('Try again')));

            // Then - linux-2 runs
            await visitor.see(heading('linux-2 running'));
        });
        expect(result.errors.text).toBe('');
    });
});
