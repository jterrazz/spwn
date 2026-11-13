import { button, component, content, field, file, region } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';
import type { Route } from '../../../shell/route';

const recordingFor = await recordingsOf('stories');

function windowOver(initial: Route) {
    return pageOver(recordingFor('stories'), initial);
}

describe('item', () => {
    test("an app's page says how it is installed, what it keeps on disk, and leaves empty what no call gives", async () => {
        // Given - the window on node's page over stories
        const result = await component.render(
            windowOver({ item: 'node', page: 'catalog-apps' }),
            async (visitor) => {
                // When - apps.list has answered
                await visitor.see(region('How it is installed'));
            },
        );

        // Then - mise on every system, its path, its disk, and the rail's empty slots
        await expect(result.tree).toMatch(file('./__expected__/stories-item-node.aria.yaml'));
    });

    test("a setting's page says how it is applied, what it changes, and the machines it is for", async () => {
        // Given - the window on agent-hooks's page over stories
        const result = await component.render(
            windowOver({ item: 'agent-hooks', page: 'catalog-settings' }),
            async (visitor) => {
                // When - settings.list and settings.matrix have answered
                await visitor.see(region('What it changes'));
                await visitor.see(button('linux-1', { exact: true }));
            },
        );

        // Then - Can be undone, claude or codex, the two files it writes, its four VMs
        await expect(result.tree).toMatch(
            file('./__expected__/stories-item-agent-hooks.aria.yaml'),
        );
    });

    test('a shelf row opens its item, and a chip opens the item it names', async () => {
        // Given - the window on Settings over stories
        const result = await component.render(
            windowOver({ page: 'catalog-settings' }),
            async (visitor) => {
                await visitor.fill(field('Search the catalog'), 'agent-hooks');

                // When - agent-hooks's row is chosen, then claude among what it needs
                await visitor.click(button('agent-hooks', { exact: false }));
                await visitor.click(button('claude', { exact: true }));

                // Then - claude's page, a script fetched from its address
                await visitor.see(content('https://claude.ai/install.sh'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a name the catalog does not hold says so', async () => {
        // Given - the window on a package no list carries
        const result = await component.render(
            windowOver({ item: 'nope', page: 'catalog-apps' }),
            async (visitor) => {
                // Then - the page says the catalog holds none
                await visitor.see(content('No nope in the catalog.'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
