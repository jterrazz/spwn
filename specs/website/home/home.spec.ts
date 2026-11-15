import { button, content, dialog, disabled, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const FIRST_RUN = '/?world=bare-mac&page=home';
const TWO_MACS = '/?world=two-macs&page=home';
const needsYou = region('Needs you');
const canWait = region('Can wait');
const memory = region('Memory');
const startHere = region('Start here');

describe('home', () => {
    test.todo(
        'H1 waits for a recorded world whose lines can all wait: two-macs and stories each hold a line that is broken, so the verdict never reads Everything is fine. on Home',
    );

    test('H2 the lines that need me are counted by the verdict and offered above everything else on Home', async () => {
        // Given - the window on Home over two-macs, where npm's sign-in expired and a background task fails
        const result = await website.visit(TWO_MACS, async (visitor) => {
            // When - I open the window
            // Then - the verdict counts the two lines, and Needs you offers each its one verb
            await visitor.see(heading('2 things need you.'));
            await visitor.see(heading('Needs you 2'));
            await visitor.see(within(needsYou, button('Copy npm login')));
            await visitor.see(within(needsYou, button('Remove com.jterrazz.brain.x-snapshot')));
        });
        expect(result.errors.text).toBe('');
    });

    test('H2 every line the engine says can wait is counted in the Inbox and offers its one verb', async () => {
        // Given - the Inbox over two-macs, whose engine lists seven lines that can wait
        const result = await website.visit('/?world=two-macs&page=inbox', async (visitor) => {
            // When - I open the window on the Inbox
            await visitor.see(canWait);

            // Then - the panel counts the lines, and each offers one verb of its own
            await visitor.see(heading('Can wait 7'));
            await visitor.see(within(canWait, button('Update mise')));
            await visitor.see(within(canWait, button('Update brew')));
            await visitor.see(within(canWait, button('Move ~/.tart')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'H3 waits for a far Mac whose overview is recorded: stories gives infra on mac-mini a failing check, but the recorder answers a far Mac with This Mac overview, which lists no line for infra',
    );

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
            // Then - there is none
            await visitor.gone(button('Stop'));
            await visitor.gone(button('Delete'));
        });
        expect(result.errors.text).toBe('');
    });

    test('H11 the head says this Mac is ready, with a Start here of three ways in', async () => {
        // Given - the window on Home over one-mac, a Mac with no machine yet
        const result = await website.visit('/?world=one-mac&page=home', async (visitor) => {
            // When - I open Home
            // Then - the engine's verdict says it, and Start here offers its three ways
            await visitor.see(heading('This Mac is ready.'));
            await visitor.see(within(startHere, button('New machine, a workspace')));
            await visitor.see(within(startHere, button('Open the file')));
            await visitor.see(within(startHere, button('Open Catalog')));
        });
        expect(result.errors.text).toBe('');
    });

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
