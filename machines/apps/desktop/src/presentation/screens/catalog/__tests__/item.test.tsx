import { button, component, content, field, file, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import type { Route } from '../../../navigation/route';

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

    test("an app's rail names the machine files that declare it and the machines that carry it", async () => {
        // Given - the window on gh's page over stories, listed by macbook-pro's file and carried by three machines
        const result = await component.render(
            windowOver({ item: 'gh', page: 'catalog-apps' }),
            async (visitor) => {
                const rail = region('gh in the catalog');

                // When - machinefile.files and apps.matrix have answered
                await visitor.see(within(rail, content('3 machines')));
            },
        );

        // Then - the files that list it are links, and the machines that carry it are chips
        await expect(result.tree).toMatch(file('./__expected__/stories-item-gh.aria.yaml'));
    });

    test("a setting's page says how it is applied, what it changes, and the machines it is for", async () => {
        // Given - the window on autologin's page over stories, a setting the world records as reversible, writing two files, for a Mac and held by mac-mini alone
        const result = await component.render(
            windowOver({ item: 'autologin', page: 'catalog-settings' }),
            async (visitor) => {
                // When - settings.list and settings.matrix have answered
                await visitor.see(region('What it changes'));
                await visitor.see(
                    within(region('autologin in the catalog'), button('mac-mini', { exact: true })),
                );
            },
        );

        // Then - Can be undone, the two files it writes, the machine that holds it
        await expect(result.tree).toMatch(file('./__expected__/stories-item-autologin.aria.yaml'));
    });

    test('a shelf row opens its item, and a chip opens the item it names', async () => {
        // Given - the window on Settings over stories
        const result = await component.render(
            windowOver({ page: 'catalog-settings' }),
            async (visitor) => {
                await visitor.click(button('Everything', { exact: false }));
                await visitor.fill(field('Search the catalog'), 'github-login');

                // When - github-login's row, which no machine holds and which requires gh, is chosen, then gh among what it needs
                await visitor.click(button('github-login', { exact: false }));
                await visitor.click(button('gh', { exact: true }));

                // Then - gh's page
                await visitor.see(content('GitHub CLI for repository management'));
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
