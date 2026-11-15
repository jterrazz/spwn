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
            // When - machinefile.files and machinefile.drift have answered
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

    test('a Mac behind its file shows what is outdated, each file tab in its layer colour', async () => {
        // Given - the window on Machine files over stories-file, where gh and ghostty are behind
        const result = await component.render(windowOver('stories-file'), async (visitor) => {
            // When - the drift has answered
            await visitor.see(content('ghostty'));
        });

        // Then - the outdated lines, and no panel the mockup does not draw
        await expect(result.tree).toMatch(file('./__expected__/stories-files.aria.yaml'));
    });

    test('every file is a tab under the caption of its shelf, in the engine words', async () => {
        // Given - the window on Machine files over stories-file, which holds yours, a mixin and shipped files
        const result = await component.render(windowOver('stories-file'), async (visitor) => {
            await visitor.see(content('ghostty'));

            // Then - one group per shelf, each under its caption
            await visitor.see(content('Your machines'));
            await visitor.see(content('Mixins'));
            await visitor.see(content('Shipped with station'));
            await visitor.see(button('station-linux'));
        });
        expect(result.errors.text).toBe('');
    });

    test('a mixin chosen says it makes no machine and lists the files that extend it', async () => {
        // Given - the window on Machine files over stories-file
        const result = await component.render(windowOver('stories-file'), async (visitor) => {
            await visitor.see(content('ghostty'));

            // When - the mixin the file extends is chosen from its facts
            await visitor.click(
                within(region('macbook-pro.machine.yaml'), button('jterrazz-base')),
            );

            // Then - it makes no machine, and the file that extends it is listed
            await visitor.see(
                content('A mixin makes no machine by itself: it is a piece other files extend'),
            );
            await visitor.see(within(region('Files that extend it'), content('macbook-pro')));
            await visitor.gone(button('New machine'));
        });
        expect(result.errors.text).toBe('');
    });

    test('a file that makes a workspace says what it makes, its size and where it starts from', async () => {
        // Given - the window on Machine files over stories-file
        const result = await component.render(windowOver('stories-file'), async (visitor) => {
            await visitor.see(content('ghostty'));

            // When - the shipped workspace file station-linux is chosen
            await visitor.click(button('station-linux'));

            // Then - the engine's word for what it makes, its size, and the image it starts from
            const head = region('station-linux.machine.yaml');
            await visitor.see(within(head, content('Workspace')));
            await visitor.see(within(head, content('4 CPU · 8G · 20G')));
            await visitor.see(within(head, content('ghcr.io/cirruslabs/ubuntu:24.04')));
        });
        expect(result.errors.text).toBe('');
    });

    test('the machines that follow a file are rows, each with its count of differences, under the engine verdict', async () => {
        // Given - the window on Machine files over stories-file, where two machines follow linux-dev and differ
        const result = await component.render(windowOver('stories-file'), async (visitor) => {
            await visitor.see(content('ghostty'));

            // When - linux-dev is chosen, its tab flagged by the engine's verdict
            await visitor.click(button('linux-dev', { exact: false }));

            // Then - the engine's verdict on its followers, and each follower with how many lines differ
            const followers = region('Machines that follow it');
            await visitor.see(within(followers, content('2 machines differ.')));
            await visitor.see(within(followers, content('agent-7f')));
            await visitor.see(within(followers, content('2 differences')));
            await visitor.see(within(followers, content('8 differences')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a file no machine follows says so, in the engine words', async () => {
        // Given - the window on Machine files over stories-file
        const result = await component.render(windowOver('stories-file'), async (visitor) => {
            await visitor.see(content('ghostty'));

            // When - the shipped file station-macos is chosen
            await visitor.click(button('station-macos'));

            // Then - the panel carries the engine's sentence alone, an empty list is no gap
            const followers = region('Machines that follow it');
            await visitor.see(within(followers, content('No machine follows it.')));
            await visitor.gone(within(followers, content('Not given yet')));
        });
        expect(result.errors.text).toBe('');
    });

    test('the YAML is the exact text of the file, and its own changes are a second tab', async () => {
        // Given - the window on Machine files over stories-file, on the file macbook-pro that has own changes
        const result = await component.render(windowOver('stories-file'), async (visitor) => {
            await visitor.see(content('ghostty'));
            await visitor.see(region("This Mac's own changes"));

            // When - YAML is chosen
            await visitor.click(within(region('macbook-pro.machine.yaml'), button('YAML')));

            // Then - the text of the file is read only, and a tab opens the changes file
            await visitor.see(region('File text'));
            await visitor.click(button('This Mac'));
            await visitor.see(region('macbook-pro.local.yaml'));
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
