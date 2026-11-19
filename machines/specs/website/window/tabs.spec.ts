import { content, heading, main, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const TABS = ['Overview', 'Activity', 'Packages', 'Ports', 'Sign-ins', 'Storage'];

describe('tabs', () => {
    test('A6 a Mac carries the six tabs', async () => {
        // Given - the window on This Mac's page over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Activity',
            async (visitor) => {
                // When - its tabs are read
                // Then - the six are there
                for (const tab of TABS) {
                    await visitor.see(within(main(), content(tab)));
                }
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('A6 a workspace carries the six tabs', async () => {
        // Given - the window on linux-1's page over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Activity',
            async (visitor) => {
                // When - its tabs are read
                // Then - the six are there
                for (const tab of TABS) {
                    await visitor.see(within(main(), content(tab)));
                }
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('A6 a server carries the tabs of its sections: Overview, Activity and Ports', async () => {
        // Given - the window on infra's page over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=infra&tab=Activity',
            async (visitor) => {
                // When - its tabs are read
                // Then - three are there, and none of Packages, Sign-ins and Storage: its engine has no such section
                for (const tab of ['Overview', 'Activity', 'Ports']) {
                    await visitor.see(within(main(), content(tab)));
                }
                for (const tab of ['Packages', 'Sign-ins', 'Storage']) {
                    await visitor.gone(within(main(), content(tab)));
                }
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('A6 a tab with nothing in it still shows its list', async () => {
        // Given - the Ports tab of This Mac over two-macs, which lists nothing
        const result = await website.visit(
            '/?world=two-macs&machine=macbook-pro&tab=Ports',
            async (visitor) => {
                // When - the tab is read
                // Then - its list says it holds nothing
                const listening = region('What listens on this Mac');
                await visitor.see(within(listening, content('Nothing listed on macbook-pro.')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('A6 a tab with nothing in it says so in its head', async () => {
        // Given - the Ports tab of mac-mini over two-macs, where its ports.list answers that nothing listens
        const result = await website.visit(
            '/?world=two-macs&machine=mac-mini&tab=Ports',
            async (visitor) => {
                // When - the tab is read
                // Then - its head carries the engine's verdict and its line
                await visitor.see(heading('Nothing listens on this machine.'));
                await visitor.see(content('A port is listed as soon as something listens on it.'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
