import { button, content, dialog, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const FIRST_RUN = '/?world=bare-mac&page=home';
const TWO_MACS = '/?world=two-macs&page=home';
const needsYou = region('Needs you');
const canWait = region('Upkeep');
const STORIES = '/?world=stories&page=home';
const machines = region('Machines');
const startHere = region('Start here');
const MINI_CARD = 'mac-mini · 3 machines · 1 machine down';

describe('home', () => {
    test('H1 the verdict reads Everything is fine. and nothing on Home needs me', async () => {
        // Given - the window on Home over calm, where every line the engine lists can wait
        const result = await website.visit('/?world=calm&page=home', async (visitor) => {
            // When - I open the window
            // Then - the verdict says all is well and counts the machines, and no Needs you is offered
            await visitor.see(heading('Everything is fine.'));
            await visitor.see(content('2 machines on macbook-pro'));
            await visitor.gone(needsYou);
        });
        expect(result.errors.text).toBe('');
        expect(result.content.text).not.toContain('need you');
    });

    test('H2 the lines that need me are counted by the verdict and offered above everything else on Home', async () => {
        // Given - the window on Home over two-macs, where a sign-in is broken and a background task fails
        const result = await website.visit(TWO_MACS, async (visitor) => {
            // When - I open the window
            // Then - the verdict counts the two, and Needs you offers each its one verb
            await visitor.see(heading('2 things need you.'));
            await visitor.see(heading('Needs you 2'));
            await visitor.see(within(needsYou, button('Fix npm')));
            await visitor.see(within(needsYou, button('Remove com.example.brain.x-snapshot')));
        });
        expect(result.errors.text).toBe('');
    });

    test('H2 every line the engine says can wait is counted in the Inbox and offers its one verb', async () => {
        // Given - the Inbox over two-macs, whose engine lists seven lines that can wait
        const result = await website.visit('/?world=two-macs&page=inbox', async (visitor) => {
            // When - I open the window on the Inbox
            await visitor.see(canWait);

            // Then - the panel counts the lines, and each offers one verb of its own
            await visitor.see(heading('Upkeep 7'));
            await visitor.see(within(canWait, button('Update mise')));
            await visitor.see(within(canWait, button('Update brew')));
            await visitor.see(within(canWait, button('Move ~/.tart')));
        });
        expect(result.errors.text).toBe('');
    });

    test('H3 a server down is named by the verdict, and is the line Needs you holds', async () => {
        // Given - the window over standings, where the server loop-1 is down in a restart loop and restart-1 can wait
        const result = await website.visit('/?world=standings&page=home', async (visitor) => {
            // When - I open the window
            // Then - the verdict names loop-1, and Needs you holds its line alone, restart-1's left to Upkeep
            await visitor.see(heading('loop-1 is down.'));
            await visitor.see(heading('Needs you 1'));
            const loop = content('loop-1 is in a loop: ↺ 5 in 10 min', { exact: false });
            await visitor.see(within(needsYou, loop));
            await visitor.gone(within(needsYou, content('restart-1', { exact: false })));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        "H3 waits for the need of a server down to name its call, machines.restart (standings' loop-1 names none, and machines.list lists it no gesture)",
    );

    test("H4 the Machines board reads each Mac's free GB, and what each machine holds", async () => {
        // Given - the window on Home over two-macs, where linux-1 holds 8 GB of macbook-pro's 16 and infra 24 of mac-mini's 64, neither use measured
        const result = await website.visit(TWO_MACS, async (visitor) => {
            // When - I look at the Machines board
            await visitor.see(within(machines, button('macbook-pro')));

            // Then - each Mac's line reads its free, and each machine's line its share
            await visitor.see(within(machines, content('4 free')));
            await visitor.see(within(machines, content('36 free')));
            await visitor.see(within(machines, content('of 8')));
            await visitor.see(within(machines, content('of 24')));
        });
        expect(result.errors.text).toBe('');
    });

    test('H5 a machine reads what it uses of what it holds', async () => {
        // Given - the window on Home over stories, where agent-7f holds 8 GB and uses 5.3, infra holds 24 and uses 16
        const result = await website.visit(STORIES, async (visitor) => {
            // When - I look at its line
            await visitor.see(within(machines, button('agent-7f')));

            // Then - each figure is its use of its share
            await visitor.see(within(machines, content('16 / 24')));
        });
        expect(result.content.text).toContain('5.3 / 8');
        expect(result.errors.text).toBe('');
    });

    test('H6 a machine working hard reads its CPU on its line of the Machines board', async () => {
        // Given - the window on Home over working-hard, where linux-1 holds every core of macbook-pro and its process takes 92 % of it
        const result = await website.visit('/?world=working-hard&page=home', async (visitor) => {
            // When - I look at the Machines board
            await visitor.see(within(machines, button('linux-1')));

            // Then - its line reads the share, and its Mac's the whole Mac's
            await visitor.see(within(machines, content('92 %')));
            await visitor.see(within(machines, content('95 %')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'H6 waits for a recording whose activity.read answers twice apart, so the figure moves and its line stays',
    );

    test.todo("H9 waits for the engine's kept journal (drawn on a provisional copy)");

    test('H10 Home offers no Stop and no Delete, only a line may offer one', async () => {
        // Given - the window on Home over two-macs, a running workspace on it
        const result = await website.visit(TWO_MACS, async (visitor) => {
            await visitor.see(within(machines, button('linux-1')));

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
            await visitor.see(dialog('New machine on macbook-pro'));
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
            await visitor.see(heading('no Machinefile named macbook-pro.'));
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

    test("H12 a Mac's card keeps its machines' lines alone, and pressed again shows them all", async () => {
        // Given - the window on Home over stories, where macbook-pro holds agent-7f and mac-mini holds infra
        const result = await website.visit(STORIES, async (visitor) => {
            await visitor.see(within(machines, button('agent-7f')));

            // When - I press the mini's card
            await visitor.click(button(MINI_CARD));

            // Then - the mini's lines alone stay, until its card is pressed again
            await visitor.see(within(machines, button('infra')));
            await visitor.gone(within(machines, button('agent-7f')));
            await visitor.click(button(MINI_CARD));
            await visitor.see(within(machines, button('agent-7f')));
        });
        expect(result.errors.text).toBe('');
    });

    test("H13 a machine's line opens on the engine's sentence, and its Open leads to the machine", async () => {
        // Given - the window on Home over stories, where infra's head says it is down
        const result = await website.visit(STORIES, async (visitor) => {
            await visitor.see(within(machines, button('infra')));

            // When - I open its line
            await visitor.click(within(machines, button('infra')));

            // Then - its line says the engine's sentence and line, and its Open leads to its page
            await visitor.see(within(machines, content('infra is down.')));
            await visitor.see(within(machines, content('Its check fails since 10:18.')));
            await visitor.click(within(machines, button('Open infra')));
            await visitor.see(heading('infra is down.'));
        });
        expect(result.errors.text).toBe('');
    });

    test('H14 a machine down reads down on its line, and its Mac card counts it', async () => {
        // Given - the window on Home over standings, where loop-1 on macbook-pro is down in a restart loop, restart-1 restarting and suspended-1 suspended
        const result = await website.visit('/?world=standings&page=home', async (visitor) => {
            // When - I look at the Machines board
            await visitor.see(within(machines, button('loop-1')));

            // Then - each line reads its own standing, and its Mac's card says one machine down, beside the mini that holds none
            await visitor.see(within(machines, content('down')));
            await visitor.see(within(machines, content('restarting')));
            await visitor.see(within(machines, content('suspended')));
            await visitor.see(button('macbook-pro · 3 machines · 1 machine down'));
            await visitor.see(button('mac-mini · no machine yet'));
        });
        expect(result.errors.text).toBe('');
    });
});
