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

describe('shelf', () => {
    test('the Skills shelf lists the agent skills alone', async () => {
        // Given - the window on Skills over stories
        const result = await component.render(
            windowOver('stories', { page: 'catalog-skills' }),
            async (visitor) => {
                // When - apps.list has answered
                await visitor.see(content('agent-browser-skill'));
            },
        );

        // Then - the agent skills, and no other app
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

    test('a description is drawn whole, on a second line when it is long, never cut', async () => {
        // Given - the window on Apps over stories, its descriptions of every length
        let cut: string[] = ['not read'];
        await component.render(windowOver('stories', { page: 'catalog-apps' }), async (visitor) => {
            // When - apps.list has answered
            await visitor.see(content('gh'));
            cut = [...document.querySelectorAll('div[class*="_cell_"]')]
                .filter((cell) => cell.scrollWidth > cell.clientWidth)
                .map((cell) => cell.textContent ?? '');
        });

        // Then - no description overflows its column
        expect(cut).toStrictEqual([]);
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
                await visitor.see(content('0 of 4 machines OK'));
            },
        );

        // Then - each setting with the matrix's count, the Checked settings, sent with no package, left off
        expect(result.tree.text).not.toContain('filevault');
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
