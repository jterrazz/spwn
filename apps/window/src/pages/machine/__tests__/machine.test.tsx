import { clock, component, content, file, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf(
    'creating',
    'machinefile-differs',
    'machinefile-kept',
    'stories',
    'two-macs',
);

function overviewOver(world: string, machine = 'jterrazz-macbook-pro') {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Overview' });
}

describe('machine', () => {
    test('a machine being created reads Creating, then its step, its position and how many there are', async () => {
        // Given - This Mac's Overview over creating, linux-4 booting and linux-5 building its image
        const result = await component.render(overviewOver('creating'), async (visitor) => {
            // When - its machines have been read
            await visitor.see(content('linux-5'));
        });

        // Then - each row says the engine's word and its birth's numbers, no state of the window's own
        expect(result.content.text).toContain('Creating · image 1 of 5');
        expect(result.content.text).toContain('Creating · boot 2 of 4');
        await expect(result.tree).toMatch(file('./__expected__/creating-mac-overview.aria.yaml'));
    });

    test('the machine being created heads its Overview with its step, and says what it is for', async () => {
        // Given - linux-5's Overview over creating, its image being built
        const result = await component.render(
            overviewOver('creating', 'linux-5'),
            async (visitor) => {
                // When - its overview has been read
                await visitor.see(content('Why it exists'));

                // Then - the head's line is its birth
                await visitor.see(content('Creating · image 1 of 5'));
            },
        );

        // Then - the purpose and the creator are the engine's
        expect(result.content.text).toContain('Lot shell, one block per package');
        expect(result.content.text).toContain('by studio-os-fe');
    });

    test("a Mac that holds its file and one whose file asks more each say the engine's verdict of the file it follows", async () => {
        // Given - This Mac's Overview over machinefile-kept, then over machinefile-differs
        const kept = await component.render(overviewOver('machinefile-kept'), async (visitor) => {
            await visitor.see(content('Matches jterrazz-macbook-pro.'));
        });
        const differs = await component.render(
            overviewOver('machinefile-differs'),
            async (visitor) => {
                // When - both overview.read and machinefile.drift answers have settled
                await visitor.see(content('2 differences from jterrazz-macbook-pro.'));
            },
        );

        // Then - each Follows row carries the drift's own sentence and tone, the window counting nothing
        await expect(kept.tree).toMatch(file('./__expected__/machinefile-overview.aria.yaml'));
        await expect(differs.tree).toMatch(
            file('./__expected__/machinefile-overview-differs.aria.yaml'),
        );
    });

    test("a workspace's Overview opens on its tiles, then its file, ports and accounts beside why it exists", async () => {
        // Given - linux-1's Overview over two-macs, a workspace running
        const result = await component.render(
            overviewOver('two-macs', 'linux-1'),
            async (visitor) => {
                // When - its activity and its file have been read
                await visitor.see(content('of 4 GB'));
                await visitor.see(content('Why it exists'));
                await visitor.see(content('Its file'));
            },
        );

        // Then - CPU, memory and disk are its numbers, who made it the engine's, what it does not give said not given
        expect(result.content.text).toContain('by you');
        await expect(result.tree).toMatch(
            file('./__expected__/two-macs-linux-1-overview.aria.yaml'),
        );
    });

    test("a VM's Overview lists none of its needs: they wait in the Inbox", async () => {
        // Given - linux-1's Overview over stories, a workspace whose overview.read names needs of its own
        const result = await component.render(
            overviewOver('stories', 'linux-1'),
            async (visitor) => {
                // When - its overview and its file have been read
                await visitor.see(content('No verdict from the engine.'));
                await visitor.see(content('Why it exists'));
                await visitor.see(content('8 differences from linux-dev.'));
            },
        );

        // Then - no panel of needs is drawn on it, and its file is the mockup's rows, Packages holding the settings
        expect(result.tree.text).not.toContain('Needs you');
        expect(result.content.text).toContain('what is installed, and its settings');
    });

    test("a server's Overview draws its health, its check's latency and its last days", async () => {
        // Given - infra's Overview over two-macs, a server on the other Mac, read 3 s after its check
        using _ = clock.at('2026-11-04T10:00:07Z');
        const result = await component.render(
            overviewOver('two-macs', 'infra'),
            async (visitor) => {
                // When - its health has been drawn from machines.list
                await visitor.see(content('Its address'));
                await visitor.see(content('41 ms now'));
            },
        );

        // Then - its state, address, SSH and check aged from the read, and the latency of each check kept
        expect(result.tree.text).toContain('checked 3 s ago');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-infra-overview.aria.yaml'));
    });

    test("a Mac's Overview draws its own measures in the same tiles, then its machines and its file", async () => {
        // Given - This Mac's Overview over two-macs
        const result = await component.render(overviewOver('two-macs'), async (visitor) => {
            // When - its activity and its file have been read
            await visitor.see(content('of 16 GB'));
            await visitor.see(content('Follows jterrazz-macbook-pro'));
        });

        // Then - the tiles hold the Mac's numbers, its machines listed, the file it follows named
        await expect(result.tree).toMatch(file('./__expected__/two-macs-mac-overview.aria.yaml'));
    });

    test("a Mac's machine reads its state and how long it has been up as one string, at the row's end", async () => {
        // Given - macbook-pro's Overview over stories, linux-1 running since its boot
        const result = await component.render(
            overviewOver('stories', 'macbook-pro'),
            async (visitor) => {
                // When - its machines are listed
                await visitor.see(heading('Its machines'));
            },
        );

        // Then - Running and its uptime share the state's place, as the mockup's "Running · up 6 d"
        expect(result.html.text).toMatch(/>Running · up \d+ (?:min|h|d)</u);
    });
});
