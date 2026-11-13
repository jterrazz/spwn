import { component, content, file, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf(
    'machinefile-differs',
    'machinefile-kept',
    'stories',
    'two-macs',
);

function overviewOver(world: string, machine = 'jterrazz-macbook-pro') {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Overview' });
}

describe('machine', () => {
    test('a Mac that holds its file and one whose file asks more read the same, the engine giving the Overview no file verdict', async () => {
        // Given - This Mac's Overview over machinefile-kept, then over machinefile-differs
        const kept = await component.render(overviewOver('machinefile-kept'), async (visitor) => {
            await visitor.see(content('No verdict from the engine.'));
            await visitor.see(content('Follows jterrazz-macbook-pro'));
        });
        const differs = await component.render(
            overviewOver('machinefile-differs'),
            async (visitor) => {
                // When - both overview.read and machinefile.drift answers have settled
                await visitor.see(content('No verdict from the engine.'));
                await visitor.see(content('Follows jterrazz-macbook-pro'));
            },
        );

        // Then - one tree for both: the Overview names the file it follows, never a count of its differences
        await expect(kept.tree).toMatch(file('./__expected__/machinefile-overview.aria.yaml'));
        expect(differs.tree.text).toBe(kept.tree.text);
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

        // Then - CPU, memory and disk are its numbers, what the engine does not give is said not given
        await expect(result.tree).toMatch(
            file('./__expected__/two-macs-linux-1-overview.aria.yaml'),
        );
    });

    test("a VM's Overview lists none of its needs: they wait in the Inbox", async () => {
        // Given - linux-1's Overview over stories, a workspace whose overview.read names needs of its own
        const result = await component.render(
            overviewOver('stories', 'linux-1'),
            async (visitor) => {
                // When - its overview has been read
                await visitor.see(content('No verdict from the engine.'));
                await visitor.see(content('Why it exists'));
            },
        );

        // Then - no panel of needs is drawn on it
        expect(result.tree.text).not.toContain('Needs you');
    });

    test("a server's Overview draws its health, its check's latency and its last days", async () => {
        // Given - infra's Overview over two-macs, a server on the other Mac
        const result = await component.render(
            overviewOver('two-macs', 'infra'),
            async (visitor) => {
                // When - its health has been drawn from machines.list
                await visitor.see(content('Its address'));
                await visitor.see(content('41 ms now'));
            },
        );

        // Then - its state, address, SSH and check, and the latency of each check kept
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
