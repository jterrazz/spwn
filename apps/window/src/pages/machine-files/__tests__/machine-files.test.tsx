import { button, component, content, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf(
    'bare-mac',
    'machinefile-differs',
    'machinefile-kept',
    'stories-file',
    'two-macs',
);

const FILES = { page: 'machine-files' } as const;

function windowOver(world: string) {
    return pageOver(recordingFor(world), FILES);
}

describe('Machine files', () => {
    test('a Mac that matches its file shows each line present', async () => {
        // Given - the window on Machine files over machinefile-kept, This Mac holding its applied file whole
        const result = await component.render(windowOver('machinefile-kept'), async (visitor) => {
            // When - machinefile.explain and machinefile.drift have answered
            await visitor.see(content('lazygit'));
        });

        // Then - the file, its layers and This Mac's lines, and no verdict the engine did not give
        await expect(result.tree).toMatch(file('./__expected__/machinefile-kept-files.aria.yaml'));
    });

    test('a Mac whose file asks more than it holds shows what it lacks', async () => {
        // Given - the window on Machine files over machinefile-differs, whose file adds ripgrep and fd
        const result = await component.render(
            windowOver('machinefile-differs'),
            async (visitor) => {
                // When - the drift has answered
                await visitor.see(content('ripgrep'));
            },
        );

        // Then - the lines the Mac lacks, in the engine words
        await expect(result.tree).toMatch(
            file('./__expected__/machinefile-differs-files.aria.yaml'),
        );
    });

    test('a Mac behind its file shows what is outdated and what applying would ask', async () => {
        // Given - the window on Machine files over stories-file, where gh and ghostty are behind
        const result = await component.render(windowOver('stories-file'), async (visitor) => {
            // When - the drift has answered
            await visitor.see(content('ghostty'));
        });

        // Then - the outdated lines and the password homebrew asks
        await expect(result.tree).toMatch(file('./__expected__/stories-files.aria.yaml'));
    });

    test('a mixin chosen is refused in the engine words', async () => {
        // Given - the window on Machine files over stories-file
        const result = await component.render(windowOver('stories-file'), async (visitor) => {
            await visitor.see(content('ghostty'));

            // When - the mixin the file extends is chosen from its facts
            await visitor.click(
                within(region('macbook-pro.machine.yaml'), button('jterrazz-base')),
            );

            // Then - the engine refuses to read a machine against a mixin
            await visitor.see(
                content('Refused · extend it from a Machinefile that says its machine'),
            );
        });
        expect(result.errors.text).toBe('');
    });

    test('a route that names a file opens that file, not the one This Mac follows', async () => {
        // Given - the window opened on Machine files over stories-file, its route naming the mixin jterrazz-base
        const result = await component.render(
            pageOver(recordingFor('stories-file'), {
                file: 'jterrazz-base',
                page: 'machine-files',
            }),
            async (visitor) => {
                // When - the page has read the file it names
                // Then - the head is the mixin's, and the engine refuses to read This Mac against it
                await visitor.see(heading('jterrazz-base.machine.yaml'));
                await visitor.see(
                    content('Refused · extend it from a Machinefile that says its machine'),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a chapter no call carries says it is not given yet', async () => {
        // Given - the window on Machine files over two-macs
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.see(content('lazygit'));

            // When - the Access chapter is chosen
            await visitor.click(button('Access'));

            // Then - the panel says what is not given yet
            await visitor.see(content('The accounts it shares and the ports it opens'));
        });
        expect(result.errors.text).toBe('');
    });

    test('a Mac with no machine file shows the engine refusal and its gesture', async () => {
        // Given - the window on Machine files over bare-mac, which follows no file
        const result = await component.render(windowOver('bare-mac'), async (visitor) => {
            // When - machinefile.explain has refused
            await visitor.see(heading('no Machinefile named jterrazz-macbook-pro.'));
        });

        // Then - the page is the refusal, with the hint the engine gave
        await expect(result.tree).toMatch(file('./__expected__/bare-mac-files.aria.yaml'));
    });
});
