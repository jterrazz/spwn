import { button, dialog, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const FIRST_RUN = '/?world=bare-mac&page=home';
const startHere = region('Start here');

describe('home', () => {
    test.todo('H1 waits for a verdict and its line on overview.read');

    test.todo('H2 lot 6');

    test.todo("H3 waits for a server's health verdict on machines.list");

    test.todo('H4 lot 6');

    test.todo('H5 waits for what each machine uses, on MachinesShare');

    test.todo('H6 lot 6');

    test.todo(
        'H7 waits for a recording of a create with a purpose, whose reads say Creating, then Running',
    );

    test.todo('H8 lot 6');

    test.todo('H9 waits for a kept journal, read by day');

    test.todo('H10 lot 6');

    test.todo('H11 waits for a verdict and its line on overview.read');

    test('H11 Start here opens the New machine sheet on this Mac, the only Mac', async () => {
        // Given - the window on Home over bare-mac, which lists no machine
        const result = await website.visit(FIRST_RUN, async (visitor) => {
            await visitor.see(startHere);

            // When - the first way in, a workspace, is pressed
            await visitor.click(within(startHere, button('New machine, a workspace')));

            // Then - the sheet opens on this Mac
            await visitor.see(dialog('New machine on jterrazz-macbook-pro'));
        });
        expect(result.errors.text).toBe('');
    });

    test('H11 Start here opens the file this Mac follows', async () => {
        // Given - the window on Home over bare-mac
        const result = await website.visit(FIRST_RUN, async (visitor) => {
            await visitor.see(startHere);

            // When - the second way in is pressed
            await visitor.click(within(startHere, button('Open the file')));

            // Then - Machine files opens, and Start here is left behind
            await visitor.see(heading('no Machinefile named jterrazz-macbook-pro.'));
            await visitor.gone(startHere);
        });
        expect(result.errors.text).toBe('');
    });

    test('H11 Start here opens the catalog', async () => {
        // Given - the window on Home over bare-mac
        const result = await website.visit(FIRST_RUN, async (visitor) => {
            await visitor.see(startHere);

            // When - the third way in is pressed
            await visitor.click(within(startHere, button('Open Catalog')));

            // Then - the catalog's apps are listed, and Start here is left behind
            await visitor.see(region('Apps'));
            await visitor.gone(startHere);
        });
        expect(result.errors.text).toBe('');
    });
});
