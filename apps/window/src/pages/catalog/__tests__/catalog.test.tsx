import { button, component, content, field, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';
import type { Route } from '../../../shell/route';

const recordingFor = await recordingsOf('bare-mac', 'stories', 'two-macs');

function windowOver(world: string, initial: Route) {
    return pageOver(recordingFor(world), initial);
}

const category = (name: string, count: number) => button(`${name}${count}`, { exact: true });

const EVERY_PACKAGE =
    'Everything station knows how to install or set, 148 packages · choose what to browse';

describe('Catalog', () => {
    test('the catalog is a chooser of three shelves, each with the categories the engine gave', async () => {
        // Given - the window on the Catalog over stories
        const result = await component.render(
            windowOver('stories', { page: 'catalog' }),
            async (visitor) => {
                // When - apps.list and settings.list have answered
                await visitor.see(content(EVERY_PACKAGE));
            },
        );

        // Then - its title, the count of what was read, and three cards, skills among them
        await expect(result.tree).toMatch(file('./__expected__/stories-catalog.aria.yaml'));
    });

    test('a card opens its shelf', async () => {
        // Given - the window on the Catalog over stories
        const result = await component.render(
            windowOver('stories', { page: 'catalog' }),
            async (visitor) => {
                // When - the Skills card is chosen
                await visitor.click(button('Skills 17 in the catalog', { exact: false }));

                // Then - the Skills shelf lists the agent skills alone
                await visitor.see(content('agent-browser-skill'));
            },
        );
        await expect(result.tree).toMatch(file('./__expected__/stories-skills.aria.yaml'));
    });

    test('apps that need the owner show the engine state on This Mac', async () => {
        // Given - the window on Apps over stories, where gh and ghostty are behind
        const result = await component.render(
            windowOver('stories', { page: 'catalog-apps' }),
            async (visitor) => {
                // When - the Git category is chosen
                await visitor.see(content('gh'));
                await visitor.click(category('Git', 4));
            },
        );

        // Then - only Git's apps remain, gh degraded, its newest version out
        await expect(result.tree).toMatch(file('./__expected__/stories-apps-git.aria.yaml'));
    });

    test('the search narrows the shelf, and says when nothing matches', async () => {
        // Given - the window on Apps over two-macs
        const result = await component.render(
            windowOver('two-macs', { page: 'catalog-apps' }),
            async (visitor) => {
                await visitor.see(content('lazygit'));

                // When - a word no package carries is searched
                await visitor.fill(field('Search the catalog'), 'zzzz');

                // Then - the list says nothing matches
                await visitor.see(content('Nothing matches.'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('settings carry their OK count on every machine', async () => {
        // Given - the window on Settings over stories
        const result = await component.render(
            windowOver('stories', { page: 'catalog-settings' }),
            async (visitor) => {
                // When - settings.list and settings.matrix have answered
                await visitor.see(content('1 of 2 machines OK'));
            },
        );

        // Then - each setting with the matrix's count
        await expect(result.tree).toMatch(file('./__expected__/stories-settings.aria.yaml'));
    });

    test('a Mac with nothing installed lists every app not installed', async () => {
        // Given - the window on Apps over bare-mac
        const result = await component.render(
            windowOver('bare-mac', { page: 'catalog-apps' }),
            async (visitor) => {
                // When - apps.list has answered
                await visitor.see(content('homebrew'));
            },
        );

        // Then - the shelf is the engine's list, every row not installed
        await expect(result.tree).toMatch(file('./__expected__/bare-mac-apps.aria.yaml'));
    });
});
