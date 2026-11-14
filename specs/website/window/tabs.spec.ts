import { content, main, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const TABS = ['Overview', 'Activity', 'Packages', 'Ports', 'Accounts', 'Storage'];

describe('tabs', () => {
    test('A6 a Mac carries the six tabs', async () => {
        // Given - the window on This Mac's page over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-macbook-pro&tab=Activity',
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

    test('A6 a server carries the six tabs', async () => {
        // Given - the window on infra's page over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=infra&tab=Activity',
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

    test('A6 a tab with nothing in it still shows its list', async () => {
        // Given - the Ports tab of This Mac over two-macs, which lists nothing
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-macbook-pro&tab=Ports',
            async (visitor) => {
                // When - the tab is read
                // Then - its list says it holds nothing
                const listening = region('What listens on this Mac');
                await visitor.see(
                    within(listening, content('Nothing listed on jterrazz-macbook-pro.')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'A6 window differs: the head of a tab with nothing in it reads No verdict from the engine, where the row has it say the tab holds nothing',
    );
});
