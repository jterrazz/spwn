import { button, content, dialog, disabled, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const FIRST_RUN = '/?world=bare-mac&page=home';
const TWO_MACS = '/?world=two-macs&page=home';
const needsYou = region('Needs you');
const memory = region('Memory');
const startHere = region('Start here');

describe('home', () => {
    test.todo('H1 waits for a verdict and its line on overview.read');

    test('H2 every line that needs me is counted and offers its one verb', async () => {
        // Given - the window on Home over two-macs, whose engine lists nine lines that need a gesture
        const result = await website.visit(TWO_MACS, async (visitor) => {
            // When - I open the window
            await visitor.see(needsYou);

            // Then - the panel counts the lines, and each offers one verb of its own
            await visitor.see(heading('Needs you 9'));
            await visitor.see(within(needsYou, button('Copy npm login')));
            await visitor.see(within(needsYou, button('Remove com.jterrazz.brain.x-snapshot')));
            await visitor.see(within(needsYou, button('Update mise')));
            await visitor.see(within(needsYou, button('Update brew')));
            await visitor.see(within(needsYou, button('Move ~/.tart')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'H2 waits for a verdict and its line on overview.read, and for a verb that reads where a region sits',
    );

    test.todo("H3 waits for a server's health verdict on machines.list");

    test('H4 Memory reads, per Mac, the GB left and who holds what', async () => {
        // Given - the window on Home over two-macs, where linux-1 holds 8 GB of one Mac's 16 and infra 24 of the other's 64
        const result = await website.visit(TWO_MACS, async (visitor) => {
            // When - I look at Memory
            await visitor.see(memory);

            // Then - each Mac reads what it has left and what each machine holds, one square per GB
            await visitor.see(within(memory, content('one square, one GB')));
            await visitor.see(within(memory, content('4 GB free of 16')));
            await visitor.see(within(memory, content('8 linux-1')));
            await visitor.see(within(memory, content('36 GB free of 64')));
            await visitor.see(within(memory, content('24 infra')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo('H5 waits for what each machine uses, on MachinesShare');

    test.todo('H6 waits for a recording of a VM above 85 % CPU, its usage read by activity.read');

    test.todo(
        'H7 waits for a recording of a create with a purpose, whose reads say Creating, then Running',
    );

    test('H8 a preset sets the memory, the figure and what it leaves follow, and no size above what is free is offered', async () => {
        // Given - the New machine sheet opened from Home over two-macs, on the Mac with 36 GB free
        const result = await website.visit(TWO_MACS, async (visitor) => {
            await visitor.click(button('New machine'));
            const sheet = dialog('New machine on jterrazz-mac-mini');
            await visitor.see(within(sheet, content('leaves 32 GB free on jterrazz-mac-mini')));

            // When - the 8 GB preset is pressed
            await visitor.click(within(sheet, button('8 GB · workspace')));

            // Then - the figure and what it leaves follow
            await visitor.see(within(sheet, content('leaves 28 GB free on jterrazz-mac-mini')));

            // When - the Mac with 4 GB free is chosen
            await visitor.click(within(sheet, button('jterrazz-macbook-pro')));

            // Then - the sheet follows that Mac, and the sizes above its room are refused
            const onIt = dialog('New machine on jterrazz-macbook-pro');
            await visitor.see(within(onIt, content('leaves 0 GB free on this Mac')));
            const eight = within(onIt, button('8 GB · no room'));
            const twelve = within(onIt, button('12 GB · no room'));
            await visitor.see(disabled(eight));
            await visitor.see(disabled(twelve));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo('H8 waits for a verb that drags across the free squares of a grid');

    test.todo("H9 waits for the engine's kept journal (drawn on a provisional copy)");

    test('H10 Home offers no Stop and no Delete, only a line may offer one', async () => {
        // Given - the window on Home over two-macs, a running workspace on it
        const result = await website.visit(TWO_MACS, async (visitor) => {
            await visitor.see(within(region('Machines'), button('linux-1')));

            // When - I look for Stop or Delete on Home
            // Then - there is none, while a line of Needs you offers its own verb
            await visitor.gone(button('Stop'));
            await visitor.gone(button('Delete'));
            await visitor.see(within(needsYou, button('Remove com.jterrazz.brain.x-snapshot')));
        });
        expect(result.errors.text).toBe('');
    });

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
