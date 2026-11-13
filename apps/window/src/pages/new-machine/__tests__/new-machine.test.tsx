import { button, component, content, dialog, field, heading, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../engine/client';
import { EngineProvider } from '../../../engine/react';
import { replayTransport } from '../../../engine/replay';
import { recordingsOf } from '../../../fixtures/recordings';
import { Window } from '../../../shell/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('one-mac', 'two-macs');

const sheet = dialog('New machine on this Mac');

function windowOver(world: string) {
    const client = new Client(replayTransport(recordingFor(world)));
    return (
        <EngineProvider client={client}>
            <Window initial={{ page: 'home' }} pages={PAGES} />
        </EngineProvider>
    );
}

describe('New machine', () => {
    test('a workspace is created once confirmed, and its page opens when the engine answers', async () => {
        // Given - the window over two-macs, its New machine sheet open
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(button('New machine'));
            await visitor.see(within(sheet, button('station-linux')));

            // When - gate5 is named, made persistent, then created and confirmed
            await visitor.fill(within(sheet, field('Name')), 'gate5');
            await visitor.click(within(sheet, button('persistent')));
            await visitor.click(within(sheet, button('Create')));
            await visitor.click(within(dialog('Create gate5?'), button('Create')));

            // Then - the sheet closes on the engine's answer, and gate5's page opens on what the engine says of gate5
            await visitor.gone(sheet);
            await visitor.see(heading('gate5 does not answer.'));
            await visitor.gone(content('No answer from the engine', { exact: false }));
        });
        expect(result.errors.text).toBe('');
    });

    test('a name the engine finds taken is refused in the sheet, in its words', async () => {
        // Given - the New machine sheet over two-macs, where linux-1 exists
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(button('New machine'));
            await visitor.see(within(sheet, button('station-linux')));

            // When - linux-1 is asked for
            await visitor.fill(within(sheet, field('Name')), 'linux-1');
            await visitor.click(within(sheet, button('Create')));
            await visitor.click(within(dialog('Create linux-1?'), button('Create')));

            // Then - the sheet stays, with the engine's refusal, its hint, and the word of a refusal of no type
            await visitor.see(
                within(sheet, content('"linux-1" is taken (on This Mac)', { exact: false })),
            );
            await visitor.see(within(sheet, content('pick another name', { exact: false })));
            await visitor.see(within(sheet, content('Error')));
        });
        expect(result.errors.text).toBe('');
    });

    test('more memory than this Mac has left is refused by the engine', async () => {
        // Given - the New machine sheet over two-macs
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(button('New machine'));
            await visitor.see(within(sheet, button('station-linux')));

            // When - big is asked with 64 GB of memory
            await visitor.fill(within(sheet, field('Name')), 'big');
            await visitor.fill(within(sheet, field('Memory, GB')), '64');
            await visitor.click(within(sheet, button('Create')));
            await visitor.click(within(dialog('Create big?'), button('Create')));

            // Then - the engine says what is left
            await visitor.see(
                within(sheet, content('64 GB asked, 4 GB left for VMs', { exact: false })),
            );
        });
        expect(result.errors.text).toBe('');
    });

    test('a server is asked of machines.create-server, and its refusal shown', async () => {
        // Given - the New machine sheet over two-macs
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(button('New machine'));
            await visitor.see(within(sheet, button('station-linux')));

            // When - a server named web is created
            await visitor.click(within(sheet, button('Server')));
            await visitor.gone(within(sheet, button('persistent')));
            await visitor.fill(within(sheet, field('Name')), 'web');
            await visitor.click(within(sheet, button('Create')));
            await visitor.click(within(dialog('Create web?'), button('Create')));

            // Then - the engine's refusal of a server's default size
            await visitor.see(
                within(sheet, content('8 GB asked, 4 GB left for VMs', { exact: false })),
            );
        });
        expect(result.errors.text).toBe('');
    });

    test('a Mac with no image says so, and the engine refuses the create', async () => {
        // Given - the New machine sheet over one-mac, which has built no image nor any machine
        const result = await component.render(windowOver('one-mac'), async (visitor) => {
            await visitor.click(button('New machine'));

            // When - gate5 is asked for
            await visitor.fill(within(sheet, field('Name')), 'gate5');
            await visitor.click(within(sheet, button('Create')));
            await visitor.click(within(dialog('Create gate5?'), button('Create')));

            // Then - the sheet named the missing image, and the engine refuses
            await visitor.see(
                within(sheet, content('images.list names no image', { exact: false })),
            );
            await visitor.see(within(sheet, content('unknown image ""', { exact: false })));
        });
        expect(result.errors.text).toBe('');
    });
});
